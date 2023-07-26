package uniswapv3

import (
	"math/big"
	"time"
)

const (
	DexTypeUniswapV3      = "uniswapv3"
	graphSkipLimit        = 5000
	graphFirstLimit       = 1000
	defaultTokenDecimals  = 18
	defaultTokenWeight    = 50
	zeroString            = "0"
	emptyString           = ""
	graphQLRequestTimeout = 20 * time.Second
)

const (
	methodGetLiquidity                    = "liquidity"
	methodGetSlot0                        = "slot0"
	tickLensMethodGetPopulatedTicksInWord = "getPopulatedTicksInWord"
	erc20MethodBalanceOf                  = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{Swap: 125000}
)
