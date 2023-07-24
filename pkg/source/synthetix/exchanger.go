package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/Synthetixio/synthetix/blob/v2.77.0-alpha/contracts/Exchanger.sol

type Exchanger struct {
	SUSDCurrencyKey string          `json:"sUSDCurrencyKey"`
	SystemSettings  *SystemSettings `json:"systemSettings"`
	ExchangeRates   *ExchangeRates  `json:"exchangeRates"`
}

func NewExchanger(
	sUSDCurrencyKey string,
	systemSettings *SystemSettings,
	exchangeRates *ExchangeRates,
) *Exchanger {
	return &Exchanger{
		SUSDCurrencyKey: sUSDCurrencyKey,
		SystemSettings:  systemSettings,
		ExchangeRates:   exchangeRates,
	}
}

func (ex *Exchanger) GetAmountsOut(
	sourceAmount *big.Int,
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (
	amountReceived *big.Int,
	fee *big.Int,
	exchangeFeeRate *big.Int,
	err error,
) {
	return ex.getAmountsForExchange(sourceAmount, sourceCurrencyKey, destinationCurrencyKey)
}

// / @notice Calculate the exchange fee for a given source and destination currency key
// / @param sourceCurrencyKey The source currency key
// / @param destinationCurrencyKey The destination currency key
// / @return The exchange fee rate
// / @return The exchange dynamic fee rate and if rates are too volatile
func (ex *Exchanger) _feeRateForExchange(
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (feeRate *big.Int, tooVolatile bool, err error) {
	// Get the exchange fee rate as per the source currencyKey and destination currencyKey
	baseRate := new(big.Int).Add(ex.getExchangeFeeRate(sourceCurrencyKey), ex.getExchangeFeeRate(destinationCurrencyKey))

	dynamicFee, tooVolatile, err := ex._dynamicFeeRateForExchange(sourceCurrencyKey, destinationCurrencyKey)
	if err != nil {
		return
	}

	return new(big.Int).Add(baseRate, dynamicFee), tooVolatile, nil
}

func (ex *Exchanger) _dynamicFeeRateForExchange(
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (
	dynamicFee *big.Int,
	tooVolatile bool,
	err error,
) {
	config := ex.getExchangeDynamicFeeConfig()
	dynamicFeeDst, dstVolatile, err := ex._dynamicFeeRateForCurrency(destinationCurrencyKey, config)
	if err != nil {
		return
	}

	dynamicFeeSrc, srcVolatile, err := ex._dynamicFeeRateForCurrency(sourceCurrencyKey, config)
	if err != nil {
		return
	}

	dynamicFee = new(big.Int).Add(dynamicFeeDst, dynamicFeeSrc)
	// cap to maxFee
	overMax := dynamicFee.Cmp(config.MaxFee) > 0

	if overMax {
		dynamicFee = config.MaxFee
	}

	return dynamicFee, overMax || dstVolatile || srcVolatile, nil
}

// / @notice Get dynamic dynamicFee for a given currency key (SIP-184)
// / @param currencyKey The given currency key
// / @param config dynamic fee calculation configuration params
// / @return The dynamic fee and if it exceeds max dynamic fee set in config
func (ex *Exchanger) _dynamicFeeRateForCurrency(currencyKey string, config DynamicFeeConfig) (dynamicFee *big.Int, tooVolatile bool, err error) {
	// no dynamic dynamicFee for sUSD or too few rounds
	if currencyKey == ex.SUSDCurrencyKey || config.Rounds.Cmp(big.NewInt(1)) <= 0 {
		return bignumber.ZeroBI, false, nil
	}
	roundId, err := ex.ExchangeRates.getCurrentRoundId(currencyKey)
	if err != nil {
		return nil, true, err
	}

	return ex._dynamicFeeRateForCurrencyRound(currencyKey, roundId, config)
}

// / @notice Get dynamicFee for a given currency key (SIP-184)
// / @param currencyKey The given currency key
// / @param roundId The round id
// / @param config dynamic fee calculation configuration params
// / @return The dynamic fee and if it exceeds max dynamic fee set in config
func (ex *Exchanger) _dynamicFeeRateForCurrencyRound(
	currencyKey string,
	roundId *big.Int,
	config DynamicFeeConfig,
) (dynamicFee *big.Int, tooVolatile bool, err error) {
	// no dynamic dynamicFee for sUSD or too few rounds
	if currencyKey == ex.SUSDCurrencyKey || config.Rounds.Cmp(big.NewInt(1)) <= 0 {
		return bignumber.ZeroBI, false, nil
	}

	prices, _, err := ex.ExchangeRates.ratesAndUpdatedTimeForCurrencyLastNRounds(currencyKey, uint(config.Rounds.Int64()), roundId)
	if err != nil {
		return
	}

	dynamicFee = _dynamicFeeCalculation(prices, config.Threshold, config.WeightDecay)
	// cap to maxFee
	overMax := dynamicFee.Cmp(config.MaxFee) > 0
	if overMax {
		dynamicFee = config.MaxFee
	}

	return dynamicFee, overMax, nil
}

// / @notice Calculate dynamic fee according to SIP-184
// / @param prices A list of prices from the current round to the previous rounds
// / @param threshold A threshold to clip the price deviation ratop
// / @param weightDecay A weight decay constant
// / @return uint dynamic fee rate as decimal
func _dynamicFeeCalculation(
	prices []*big.Int,
	threshold *big.Int,
	weightDecay *big.Int,
) *big.Int {
	// don't underflow
	if len(prices) == 0 {
		return bignumber.ZeroBI
	}

	dynamicFee := new(big.Int).Set(bignumber.ZeroBI) // start with 0
	// go backwards in price array
	for i := len(prices) - 1; i > 0; i-- {
		// apply decay from previous round (will be 0 for first round)
		dynamicFee = multiplyDecimal(dynamicFee, weightDecay)
		// calculate price deviation
		deviation := _thresholdedAbsDeviationRatio(prices[i-1], prices[i], threshold)
		// add to total fee
		dynamicFee = new(big.Int).Add(dynamicFee, deviation)
	}

	return dynamicFee
}

// / absolute price deviation ratio used by dynamic fee calculation
// / deviationRatio = (abs(current - previous) / previous) - threshold
// / if negative, zero is returned
func _thresholdedAbsDeviationRatio(
	price *big.Int,
	previousPrice *big.Int,
	threshold *big.Int,
) *big.Int {
	if previousPrice.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI // don't divide by zero
	}
	// abs difference between prices
	var absDelta *big.Int
	if price.Cmp(previousPrice) > 0 {
		absDelta = new(big.Int).Sub(price, previousPrice)
	} else {
		absDelta = new(big.Int).Sub(previousPrice, price)
	}

	// relative to previous price
	deviationRatio := divideDecimal(absDelta, previousPrice)
	// only the positive difference from threshold
	if deviationRatio.Cmp(threshold) > 0 {
		return new(big.Int).Sub(deviationRatio, threshold)
	}

	return bignumber.ZeroBI
}

func (ex *Exchanger) getAmountsForExchange(
	sourceAmount *big.Int,
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (
	amountReceived *big.Int,
	fee *big.Int,
	exchangeFeeRate *big.Int,
	err error,
) {
	if sourceCurrencyKey != ex.SUSDCurrencyKey && ex.ExchangeRates.rateIsInvalid(sourceCurrencyKey) {
		err = ErrInvalidSrcSynth

		return
	}

	if destinationCurrencyKey != ex.SUSDCurrencyKey && ex.ExchangeRates.rateIsInvalid(destinationCurrencyKey) {
		err = ErrInvalidDestSynth

		return
	}

	exchangeFeeRate, tooVolatile, err := ex._feeRateForExchange(sourceCurrencyKey, destinationCurrencyKey)
	if err != nil {
		return
	}

	if tooVolatile {
		err = ErrExchangeRatesTooVolatile

		return
	}

	destinationAmount, _, _, err := ex.ExchangeRates.effectiveValueAndRates(sourceCurrencyKey, sourceAmount, destinationCurrencyKey)

	amountReceived = ex._deductFeesFromAmount(destinationAmount, exchangeFeeRate)
	fee = new(big.Int).Sub(destinationAmount, amountReceived)

	return
}

func (ex *Exchanger) _deductFeesFromAmount(
	destinationAmount *big.Int,
	exchangeFeeRate *big.Int,
) (amountReceived *big.Int) {
	amountReceived = multiplyDecimal(destinationAmount, new(big.Int).Sub(unit(), exchangeFeeRate))

	return
}

func (ex *Exchanger) getExchangeFeeRate(currencyKey string) *big.Int {
	return ex.SystemSettings.ExchangeFeeRate[currencyKey]
}

func (ex *Exchanger) getExchangeDynamicFeeConfig() DynamicFeeConfig {
	return *ex.SystemSettings.DynamicFeeConfig
}
