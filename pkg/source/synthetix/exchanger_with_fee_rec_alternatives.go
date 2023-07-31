package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// =============================================================================================
// Implementation of this contract
// https://github.com/Synthetixio/synthetix/blob/b04a4d2948f3a575bfe8186e99086e50dc54ef95/contracts/ExchangerWithFeeRecAlternatives.sol

type ExchangerWithFeeRecAlternatives struct {
	SUSDCurrencyKey             string
	BlockTimestamp              uint64
	LastAtomicVolume            *ExchangeVolumeAtPeriod
	AtomicMaxVolumePerBlock     *big.Int
	SystemSettings              *SystemSettings
	ExchangeRatesWithDexPricing *ExchangeRatesWithDexPricing
}

func NewExchangerWithFeeRecAlternatives(
	sUSDCurrencyKey string,
	blockTimestamp uint64,
	lastAtomicVolume *ExchangeVolumeAtPeriod,
	atomicMaxVolumePerBlock *big.Int,
	systemSettings *SystemSettings,
	exchangeRatesWithDexPricing *ExchangeRatesWithDexPricing,
) *ExchangerWithFeeRecAlternatives {
	return &ExchangerWithFeeRecAlternatives{
		SUSDCurrencyKey:             sUSDCurrencyKey,
		BlockTimestamp:              blockTimestamp,
		LastAtomicVolume:            lastAtomicVolume,
		AtomicMaxVolumePerBlock:     atomicMaxVolumePerBlock,
		SystemSettings:              systemSettings,
		ExchangeRatesWithDexPricing: exchangeRatesWithDexPricing,
	}
}

func (ex *ExchangerWithFeeRecAlternatives) GetAmountsOut(
	sourceAmount *big.Int,
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (
	amountReceived *big.Int,
	fee *big.Int,
	exchangeFeeRate *big.Int,
	err error,
) {
	if ex.ExchangeRatesWithDexPricing.SynthTooVolatileForAtomicExchanges[sourceCurrencyKey] {
		return nil, nil, nil, ErrSrcSynthTooVolatile
	}

	if ex.ExchangeRatesWithDexPricing.SynthTooVolatileForAtomicExchanges[destinationCurrencyKey] {
		return nil, nil, nil, ErrDestSynthTooVolatile
	}

	amountReceived, fee, exchangeFeeRate, systemConvertedAmount, _, _, err := ex._getAmountsForAtomicExchangeMinusFees(
		sourceAmount,
		sourceCurrencyKey,
		destinationCurrencyKey,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// Determine sUSD value (volume) of exchange
	var sourceSusdValue *big.Int
	if sourceCurrencyKey == ex.SUSDCurrencyKey {
		// Use sourceAmount directly
		// We can NOT simulate using [sourceAmountAfterSettlement] as in the contract
		// https://github.com/Synthetixio/synthetix/blob/3521196cc1a588d8419bff41d515ca4f468a3e29/contracts/ExchangerWithFeeRecAlternatives.sol#L180
		// Because in order to calculate the [sourceAmountAfterSettlement], we need from address balance, which is different for every request
		sourceSusdValue = sourceAmount
	} else if destinationCurrencyKey == ex.SUSDCurrencyKey {
		// In this case the systemConvertedAmount would be the fee-free sUSD value of the source synth
		sourceSusdValue = systemConvertedAmount
	} else {
		// Otherwise, convert source to sUSD value
		amountReceivedInUSD, sUsdFee, _, _, _, _, err :=
			ex._getAmountsForAtomicExchangeMinusFees(sourceAmount, sourceCurrencyKey, ex.SUSDCurrencyKey)
		if err != nil {
			return nil, nil, nil, err
		}

		sourceSusdValue = new(big.Int).Add(amountReceivedInUSD, sUsdFee)
	}

	if err := ex._checkAndUpdateAtomicVolume(sourceSusdValue); err != nil {
		return nil, nil, nil, err
	}

	return
}

func (ex *ExchangerWithFeeRecAlternatives) _deductFeesFromAmount(
	destinationAmount *big.Int,
	exchangeFeeRate *big.Int,
) (amountReceived *big.Int) {
	amountReceived = multiplyDecimal(destinationAmount, new(big.Int).Sub(unit(), exchangeFeeRate))

	return
}

// getSourceSUSDValue returns the volume of the trade in sUSD
func (ex *ExchangerWithFeeRecAlternatives) getSourceSUSDValue(
	sourceAmount *big.Int,
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (*big.Int, error) {
	// Determine sUSD value (volume) of exchange
	var sourceSusdValue *big.Int
	switch {
	case sourceCurrencyKey == ex.SUSDCurrencyKey:
		// Use sourceAmount directly
		// We can NOT simulate using [sourceAmountAfterSettlement] as in the contract
		// https://github.com/Synthetixio/synthetix/blob/3521196cc1a588d8419bff41d515ca4f468a3e29/contracts/ExchangerWithFeeRecAlternatives.sol#L180
		// Because in order to calculate the [sourceAmountAfterSettlement], we need from address balance, which is different for every request
		sourceSusdValue = sourceAmount
	case destinationCurrencyKey == ex.SUSDCurrencyKey:
		_, _, _, systemConvertedAmount, _, _, err := ex._getAmountsForAtomicExchangeMinusFees(
			sourceAmount,
			sourceCurrencyKey,
			destinationCurrencyKey,
		)
		if err != nil {
			return nil, err
		}

		// In this case the systemConvertedAmount would be the fee-free sUSD value of the source synth
		sourceSusdValue = systemConvertedAmount
	default:
		// Otherwise, convert source to sUSD value
		amountReceivedInUSD, sUsdFee, _, _, _, _, err :=
			ex._getAmountsForAtomicExchangeMinusFees(sourceAmount, sourceCurrencyKey, ex.SUSDCurrencyKey)
		if err != nil {
			return nil, err
		}

		sourceSusdValue = new(big.Int).Add(amountReceivedInUSD, sUsdFee)
	}

	return sourceSusdValue, nil
}

//func (ex *ExchangerWithFeeRecAlternatives) getAmountsForAtomicExchange(
//	sourceAmount *big.Int,
//	sourceCurrencyKey string,
//	destinationCurrencyKey string,
//) (
//	amountReceived *big.Int,
//	fee *big.Int,
//	exchangeFeeRate *big.Int,
//	err error,
//) {
//	amountReceived, fee, exchangeFeeRate, _, _, _, err = ex._getAmountsForAtomicExchangeMinusFees(
//		sourceAmount,
//		sourceCurrencyKey,
//		destinationCurrencyKey,
//	)
//
//	return
//}

func (ex *ExchangerWithFeeRecAlternatives) _checkAndUpdateAtomicVolume(sourceSusdValue *big.Int) error {
	if ex.LastAtomicVolume == nil {
		return ErrInvalidLastAtomicVolume
	}

	var currentVolume *big.Int

	if ex.LastAtomicVolume.Time == ex.BlockTimestamp {
		currentVolume = new(big.Int).Add(ex.LastAtomicVolume.Volume, sourceSusdValue)
	} else {
		currentVolume = sourceSusdValue
	}

	if currentVolume.Cmp(ex.getAtomicMaxVolumePerBlock()) > 0 {
		return ErrSurpassedVolumeLimit
	}

	return nil
}

func (ex *ExchangerWithFeeRecAlternatives) _feeRateForAtomicExchange(sourceCurrencyKey string, destinationCurrencyKey string) *big.Int {
	// Get the exchange fee rate as per source and destination currencyKey
	baseRate := new(big.Int).Add(ex.getAtomicExchangeFeeRate(sourceCurrencyKey), ex.getAtomicExchangeFeeRate(destinationCurrencyKey))

	if baseRate.Cmp(bignumber.ZeroBI) == 0 {
		// If no atomic rate was set, fallback to the regular exchange rate
		baseRate = new(big.Int).Add(ex.getExchangeFeeRate(sourceCurrencyKey), ex.getExchangeFeeRate(destinationCurrencyKey))
	}

	return baseRate
}

func (ex *ExchangerWithFeeRecAlternatives) _getAmountsForAtomicExchangeMinusFees(
	sourceAmount *big.Int,
	sourceCurrencyKey string,
	destinationCurrencyKey string,
) (
	amountReceived *big.Int,
	fee *big.Int,
	exchangeFeeRate *big.Int,
	systemConvertedAmount *big.Int,
	systemSourceRate *big.Int,
	systemDestinationRate *big.Int,
	err error,
) {
	var destinationAmount *big.Int

	destinationAmount, systemConvertedAmount, systemSourceRate, systemDestinationRate, err = ex.ExchangeRatesWithDexPricing.effectiveAtomicValueAndRates(sourceCurrencyKey, sourceAmount, destinationCurrencyKey)
	if err != nil {
		return
	}

	exchangeFeeRate = ex._feeRateForAtomicExchange(sourceCurrencyKey, destinationCurrencyKey)
	amountReceived = ex._deductFeesFromAmount(destinationAmount, exchangeFeeRate)
	fee = new(big.Int).Sub(destinationAmount, amountReceived)

	return
}

func (ex *ExchangerWithFeeRecAlternatives) getAtomicExchangeFeeRate(currencyKey string) *big.Int {
	return ex.SystemSettings.AtomicExchangeFeeRate[currencyKey]
}

func (ex *ExchangerWithFeeRecAlternatives) getExchangeFeeRate(currencyKey string) *big.Int {
	return ex.SystemSettings.ExchangeFeeRate[currencyKey]
}

func (ex *ExchangerWithFeeRecAlternatives) getAtomicMaxVolumePerBlock() *big.Int {
	return ex.AtomicMaxVolumePerBlock
}
