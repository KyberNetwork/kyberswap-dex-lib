package syncswapv2

var (
	DexTypeSyncSwapV2 = "syncswapv2"

	PoolTypeSyncSwapV2Classic               = "syncswapv2-classic"
	PoolTypeSyncSwapV2Stable                = "syncswapv2-stable"
	PoolTypeSyncSwapV2Aqua                  = "syncswapv2-aqua"
	poolTypeSyncSwapV2StableInContract      = 2
	poolTypeSyncSwapV2AquaInContract        = 3
	defaultTokenWeight                 uint = 50
	reserveZero                             = "0"
	addressZero                             = "0x0000000000000000000000000000000000000000"

	poolMasterMethodPoolsLength         = "poolsLength"
	poolMasterMethodPools               = "pools"
	poolMethodPoolType                  = "poolType"
	poolMethodGetAssets                 = "getAssets"
	poolMethodGetSwapFee                = "getSwapFee"
	poolMethodGetReserves               = "getReserves"
	poolMethodToken0PrecisionMultiplier = "token0PrecisionMultiplier"
	poolMethodToken1PrecisionMultiplier = "token1PrecisionMultiplier"
	poolMethodVault                     = "vault"
	poolMethodGetA                      = "getA"
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
