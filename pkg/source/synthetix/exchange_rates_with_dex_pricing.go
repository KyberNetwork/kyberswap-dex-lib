package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// =============================================================================================
// Implementation of this contract
// https://github.com/Synthetixio/synthetix/blob/b2c49e61dd9a75864d5c455311375d17f023e310/contracts/ExchangeRatesWithDexPricing.sol

type RateAndUpdatedTime struct {
	rate *big.Int
	time *big.Int
}

type ExchangeRatesWithDexPricing struct {
	SUSDCurrencyKey                    string                        `json:"sUSDCurrencyKey"`
	SystemSettings                     *SystemSettings               `json:"systemSettings"`
	Aggregators                        map[string]*ChainlinkDataFeed `json:"aggregators"`
	CurrencyKeyDecimals                map[string]uint8              `json:"currencyKeyDecimals"`
	DexPriceAggregator                 *DexPriceAggregatorUniswapV3  `json:"dexPriceAggregator"`
	SynthTooVolatileForAtomicExchanges map[string]bool               `json:"synthTooVolatileForAtomicExchange,omitempty"`
}

func NewExchangeRatesWithDexPricing(
	sUSDCurrencyKey string,
	systemSettings *SystemSettings,
	aggregators map[string]*ChainlinkDataFeed,
	currencyKeyDecimals map[string]uint8,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	synthTooVolatileForAtomicExchanges map[string]bool,
) *ExchangeRatesWithDexPricing {
	return &ExchangeRatesWithDexPricing{
		SUSDCurrencyKey:                    sUSDCurrencyKey,
		SystemSettings:                     systemSettings,
		Aggregators:                        aggregators,
		CurrencyKeyDecimals:                currencyKeyDecimals,
		DexPriceAggregator:                 dexPriceAggregator,
		SynthTooVolatileForAtomicExchanges: synthTooVolatileForAtomicExchanges,
	}
}

func (er *ExchangeRatesWithDexPricing) _formatAggregatorAnswer(currencyKey string, rate *big.Int) (*big.Int, error) {
	if rate.Cmp(bignumber.ZeroBI) < 0 {
		return nil, ErrNegativeRate
	}

	decimals := er.CurrencyKeyDecimals[currencyKey]
	result := rate
	if decimals == 0 || decimals == 18 {
		// do not convert for 0 (part of implicit interface), and not needed for 18
	} else if decimals < 18 {
		// increase precision to 18
		multiplier := bignumber.TenPowInt(18 - decimals)
		result = new(big.Int).Mul(result, multiplier)
	} else if decimals > 18 {
		// decrease precision to 18
		divisor := bignumber.TenPowInt(decimals - 18)
		result = new(big.Int).Div(result, divisor)
	}

	return result, nil
}

func (er *ExchangeRatesWithDexPricing) _getRateAndUpdatedTime(currencyKey string) (RateAndUpdatedTime, error) {
	// sUSD rate is 1.0
	if currencyKey == er.SUSDCurrencyKey {
		return RateAndUpdatedTime{
			rate: bignumber.TenPowInt(18),
			time: bignumber.ZeroBI,
		}, nil
	} else {
		aggregator := er.Aggregators[currencyKey]
		if aggregator == nil {
			return RateAndUpdatedTime{}, ErrAggregatorNotFound
		}

		answer := aggregator.LatestAnswer()
		updatedAt := aggregator.LatestUpdatedAt()

		rate, err := er._formatAggregatorAnswer(currencyKey, answer)
		if err != nil {
			return RateAndUpdatedTime{}, err
		}

		return RateAndUpdatedTime{
			rate: rate,
			time: updatedAt,
		}, nil

	}
}

//func (er *ExchangeRatesWithDexPricing) _getCurrentRoundId(currencyKey string) (*big.Int, error) {
//	if currencyKey == er.SUSDCurrencyKey {
//		return constant.Zero, nil
//	}
//
//	aggregator := er.Aggregators[currencyKey]
//	if aggregator == nil {
//		return nil, ErrAggregatorNotFound
//	}
//
//	return aggregator.LatestRound(), nil
//}

//func (er *ExchangeRatesWithDexPricing) _getRateAndTimestampAtRound(currencyKey string, roundId *big.Int) (*big.Int, *big.Int, error) {
//	// short circuit sUSD
//	if currencyKey == er.SUSDCurrencyKey {
//		// sUSD has no rounds, and 0 time is preferrable for "volatility" heuristics
//		// which are used in atomic swaps and fee reclamation
//		return unit(), constant.Zero, nil
//	}
//
//	aggregator := er.Aggregators[currencyKey]
//	if aggregator == nil {
//		return nil, nil, ErrAggregatorNotFound
//	}
//
//	_, answer, _, updatedAt, _ := aggregator.GetRoundData(roundId)
//
//	rate, err := er._formatAggregatorAnswer(currencyKey, answer)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	return rate, updatedAt, nil
//}

func (er *ExchangeRatesWithDexPricing) _getRate(currencyKey string) (*big.Int, error) {
	rateAndUpdatedTime, err := er._getRateAndUpdatedTime(currencyKey)
	if err != nil {
		return nil, err
	}

	return rateAndUpdatedTime.rate, nil
}

func (er *ExchangeRatesWithDexPricing) _effectiveValueAndRates(
	sourceCurrencyKey string,
	sourceAmount *big.Int,
	destinationCurrencyKey string,
) (value *big.Int, sourceRate *big.Int, destinationRate *big.Int, err error) {
	sourceRate, err = er._getRate(sourceCurrencyKey)
	if err != nil {
		return
	}

	// If there's no change in the currency, then just return the amount they gave us
	if sourceCurrencyKey == destinationCurrencyKey {
		destinationRate = sourceRate
		value = sourceAmount
	} else {
		// Calculate the effective value by going from source -> USD -> destination
		destinationRate, err = er._getRate(destinationCurrencyKey)
		if err != nil {
			return
		}

		// prevent divide-by 0 error (this happens if the dest is not a valid rate)
		if destinationRate.Cmp(bignumber.ZeroBI) > 0 {
			value = divideDecimalRound(multiplyDecimalRound(sourceAmount, sourceRate), destinationRate)
		}
	}

	return
}

func (er *ExchangeRatesWithDexPricing) getAtomicTwapWindow() *big.Int {
	return er.SystemSettings.AtomicTwapWindow
}

func (er *ExchangeRatesWithDexPricing) getPureChainlinkPriceForAtomicSwapsEnabled(currencyKey string) bool {
	return er.SystemSettings.PureChainlinkPriceForAtomicSwapsEnabled[currencyKey]
}

func (er *ExchangeRatesWithDexPricing) getAtomicEquivalentForDexPricing(currencyKey string) Token {
	return er.SystemSettings.AtomicEquivalentForDexPricing[currencyKey]
}

//func (er *ExchangeRatesWithDexPricing) getAtomicVolatilityConsiderationWindow(currencyKey string) *big.Int {
//	return er.SystemSettings.AtomicVolatilityConsiderationWindow[currencyKey]
//}

//func (er *ExchangeRatesWithDexPricing) getAtomicVolatilityUpdateThreshold(currencyKey string) *big.Int {
//	return er.SystemSettings.AtomicVolatilityUpdateThreshold[currencyKey]
//}

func (er *ExchangeRatesWithDexPricing) effectiveAtomicValueAndRates(
	sourceCurrencyKey string,
	sourceAmount *big.Int,
	destinationCurrencyKey string,
) (
	value *big.Int,
	systemValue *big.Int,
	systemSourceRate *big.Int,
	systemDestinationRate *big.Int,
	err error,
) {
	systemValue, systemSourceRate, systemDestinationRate, err = er._effectiveValueAndRates(
		sourceCurrencyKey,
		sourceAmount,
		destinationCurrencyKey,
	)
	if err != nil {
		return
	}

	usePureChainlinkPriceForSource := er.getPureChainlinkPriceForAtomicSwapsEnabled(sourceCurrencyKey)
	usePureChainlinkPriceForDest := er.getPureChainlinkPriceForAtomicSwapsEnabled(destinationCurrencyKey)

	var sourceRate, destRate *big.Int
	// Handle the different scenarios that may arise when trading currencies with or without the PureChainlinkPrice set.
	// outlined here: https://sips.synthetix.io/sips/sip-198/#computation-methodology-in-atomic-pricing
	if usePureChainlinkPriceForSource {
		sourceRate = systemSourceRate
	} else {
		priceFromDexAggregator, err := er._getPriceFromDexAggregator(sourceCurrencyKey, er.SUSDCurrencyKey, sourceAmount)
		if err != nil {
			return value, systemValue, systemSourceRate, systemDestinationRate, err
		}

		sourceRate = _getMinValue(systemSourceRate, priceFromDexAggregator)
	}

	if usePureChainlinkPriceForDest {
		destRate = systemDestinationRate
	} else {
		priceFromDexAggregator, err := er._getPriceFromDexAggregator(er.SUSDCurrencyKey, destinationCurrencyKey, sourceAmount)
		if err != nil {
			return value, systemValue, systemSourceRate, systemDestinationRate, err
		}

		destRate = _getMaxValue(systemDestinationRate, priceFromDexAggregator)
	}

	value = new(big.Int).Div(new(big.Int).Mul(sourceAmount, sourceRate), destRate)

	return
}

func _getMinValue(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return x
	}

	return y
}

func _getMaxValue(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return x
	}

	return y
}

func (er *ExchangeRatesWithDexPricing) _getPriceFromDexAggregator(
	sourceCurrencyKey string,
	destCurrencyKey string,
	amount *big.Int,
) (*big.Int, error) {
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrAmountZero
	}

	if sourceCurrencyKey != er.SUSDCurrencyKey && destCurrencyKey != er.SUSDCurrencyKey {
		return nil, ErrInvalidAtomicSwaps
	}

	sourceEquivalent := er.getAtomicEquivalentForDexPricing(sourceCurrencyKey)
	if eth.IsZeroAddress(sourceEquivalent.Address) {
		return nil, ErrNoAtomicEquivalentForSource
	}

	destEquivalent := er.getAtomicEquivalentForDexPricing(destCurrencyKey)
	if eth.IsZeroAddress(destEquivalent.Address) {
		return nil, ErrNoAtomicEquivalentForDest
	}

	destinationValue, err := er._dexPriceDestinationValue(sourceEquivalent, destEquivalent, amount)
	if err != nil {
		return nil, err
	}

	result := new(big.Int).Div(new(big.Int).Mul(destinationValue, unit()), amount)
	if result.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrResultZero
	}

	if destCurrencyKey == er.SUSDCurrencyKey {
		return result, nil
	}

	return divideDecimalRound(unit(), result), nil
}

func (er *ExchangeRatesWithDexPricing) _dexPriceDestinationValue(
	sourceEquivalent Token,
	destEquivalent Token,
	sourceAmount *big.Int,
) (*big.Int, error) {
	// Normalize decimals in case equivalent asset uses different decimals from internal unit
	sourceAmountInEquivalent := new(big.Int).Div(new(big.Int).Mul(sourceAmount, bignumber.TenPowInt(sourceEquivalent.Decimals)), unit())

	twapWindow := er.getAtomicTwapWindow()
	if twapWindow.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrUninitializedAtomicTwapWindow
	}

	twapValueInEquivalent, err := er.DexPriceAggregator.assetToAsset(
		sourceEquivalent.Address,
		sourceAmountInEquivalent,
		destEquivalent.Address,
		twapWindow,
	)
	if err != nil {
		return nil, err
	}

	if twapValueInEquivalent.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrDexPriceZero
	}

	// Similar to source amount, normalize decimals back to internal unit for output amount
	return new(big.Int).Div(new(big.Int).Mul(twapValueInEquivalent, unit()), bignumber.TenPowInt(destEquivalent.Decimals)), nil
}

//func (er *ExchangeRatesWithDexPricing) synthTooVolatileForAtomicExchange(currencyKey string) bool {
//	return er.SynthTooVolatileForAtomicExchanges[currencyKey]
//}
