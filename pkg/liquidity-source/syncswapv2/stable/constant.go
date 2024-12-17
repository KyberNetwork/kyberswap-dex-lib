package syncswapv2stable

import (
	"github.com/holiman/uint256"
)

var (
	DexTypeSyncSwapV2Stable                 = "syncswapv2-stable"
	PoolTypeSyncSwapV2Stable                = "syncswapv2-stable"
	poolTypeSyncSwapV2StableInContract      = 2
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
)

var (
	MaxFee = uint256.NewInt(100000)
)
