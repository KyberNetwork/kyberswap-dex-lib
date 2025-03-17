package syncswap

var (
	DexTypeSyncSwap = "syncswap"

	PoolTypeSyncSwapClassic = "syncswap-classic"
	PoolTypeSyncSwapStable  = "syncswap-stable"

	// poolTypeSyncSwapClassicInContract = 1
	poolTypeSyncSwapStableInContract = 2

	defaultTokenWeight uint = 50
	reserveZero             = "0"
	addressZero             = "0x0000000000000000000000000000000000000000"

	poolMasterMethodPoolsLength = "poolsLength"
	poolMasterMethodPools       = "pools"

	poolMethodPoolType                  = "poolType"
	poolMethodGetAssets                 = "getAssets"
	poolMethodGetSwapFee                = "getSwapFee"
	poolMethodGetReserves               = "getReserves"
	poolMethodToken0PrecisionMultiplier = "token0PrecisionMultiplier"
	poolMethodToken1PrecisionMultiplier = "token1PrecisionMultiplier"
	poolMethodVault                     = "vault"
	poolMethodBalanceOf                 = "balanceOf"
)
