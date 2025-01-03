package elastic

import (
	"math/big"
)

const (
	DexTypeElastic       = "elastic"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	defaultTokenWeight   = 50
	reserveZero          = "0"
)

const (
	methodGetLiquidityState = "getLiquidityState"
	methodGetPoolState      = "getPoolState"
	erc20MethodBalanceOf    = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{SwapBase: 100000}
)
