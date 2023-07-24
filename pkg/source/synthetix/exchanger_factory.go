package synthetix

import (
	"math/big"
)

type IExchanger interface {
	GetAmountsOut(
		sourceAmount *big.Int,
		sourceCurrencyKey string,
		destinationCurrencyKey string,
	) (
		amountReceived *big.Int,
		fee *big.Int,
		exchangeFeeRate *big.Int,
		err error,
	)
}

func GetExchanger(poolStateVersion PoolStateVersion, poolState *PoolState) IExchanger {
	if poolStateVersion == PoolStateVersionNormal {
		return NewExchanger(poolState.SUSDCurrencyKey,
			poolState.SystemSettings,
			NewExchangeRates(
				poolState.BlockTimestamp,
				poolState.SUSDCurrencyKey,
				poolState.SystemSettings,
				poolState.Aggregators,
				poolState.CurrencyKeyDecimals,
			))
	}

	return NewExchangerWithFeeRecAlternatives(
		poolState.SUSDCurrencyKey,
		poolState.BlockTimestamp,
		poolState.LastAtomicVolume,
		poolState.AtomicMaxVolumePerBlock,
		poolState.SystemSettings,
		NewExchangeRatesWithDexPricing(
			poolState.SUSDCurrencyKey,
			poolState.SystemSettings,
			poolState.Aggregators,
			poolState.CurrencyKeyDecimals,
			poolState.DexPriceAggregator,
			poolState.SynthTooVolatileForAtomicExchanges,
		))
}
