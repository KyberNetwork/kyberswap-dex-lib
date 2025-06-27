package v2

const (
	DexType = "compound-v2"

	mintGas   int64 = 300000
	redeemGas int64 = 250000

	defaultReserve = 10000000000

	cTokenMethodExchangeRateStored = "exchangeRateStored"
	cTokenMethodUnderlying         = "underlying"

	comptrollerMethodGetAllMarkets      = "getAllMarkets"
	comptrollerMethodMintGuardianPaused = "mintGuardianPaused"
)
