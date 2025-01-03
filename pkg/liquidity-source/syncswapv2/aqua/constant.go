package syncswapv2aqua

var (
	DexTypeSyncSwapV2Aqua = "syncswapv2-aqua"

	PoolTypeSyncSwapV2Aqua                = "syncswapv2-aqua"
	poolTypeSyncSwapV2AquaInContract      = 3
	defaultTokenWeight               uint = 50
	reserveZero                           = "0"
	addressZero                           = "0x0000000000000000000000000000000000000000"

	poolMethodPoolType                  = "poolType"
	poolMethodGetAssets                 = "getAssets"
	poolMethodGetReserves               = "getReserves"
	poolMethodToken0PrecisionMultiplier = "token0PrecisionMultiplier"
	poolMethodToken1PrecisionMultiplier = "token1PrecisionMultiplier"
	poolMethodVault                     = "vault"
	poolMethodGetFeeManager             = "feeManager"

	poolMethodAquaParams              = "getParams"
	poolMethodAquaPoolParams          = "poolParams"
	poolMethodAquaGetLastPrices       = "lastPrices"
	poolMethodAquaLastPricesTimestamp = "lastPricesTimestamp"
	poolMethodAquaPriceScale          = "priceScale"
	poolMethodAquaD                   = "invariantLast"
	poolMethodAquaGetSwapFee          = "getSwapFeeData"
	poolMethodAquaPriceOracle         = "cachedPriceOracle"
	poolMethodAquaTotalSupply         = "totalSupply"
	poolMethodAquaXcpProfit           = "xcpProfit"
	poolMethodAquaVirtualPrice        = "getVirtualPrice"
	poolMethodAquaRebalancingParams   = "rebalancingParams" //allowedExtraProfit + adjustmentStep + maTime
)
