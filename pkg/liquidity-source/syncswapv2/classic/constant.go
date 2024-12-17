package syncswapv2classic

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"

var (
	DexTypeSyncSwapV2Classic                 = "syncswapv2-classic"
	PoolTypeSyncSwapV2Classic                = "syncswapv2-classic"
	poolTypeSyncSwapV2ClassicInContract      = 1
	defaultTokenWeight                  uint = 50
	reserveZero                              = "0"
	addressZero                              = "0x0000000000000000000000000000000000000000"

	poolMasterMethodPoolsLength = "poolsLength"
	poolMasterMethodPools       = "pools"
	poolMethodPoolType          = "poolType"
	poolMethodGetAssets         = "getAssets"
	poolMethodGetSwapFee        = "getSwapFee"
	poolMethodGetReserves       = "getReserves"
	poolMethodVault             = "vault"
)

var (
	DefaultGas = syncswap.Gas{Swap: 300000}
)
