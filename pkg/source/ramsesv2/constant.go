package ramsesv2

import (
	"math/big"
	"time"
)

const (
	DexTypeRamsesV2       = "ramses-v2"
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
	methodCurrentFee                      = "currentFee"
	methodTickSpacing                     = "tickSpacing"
	tickLensMethodGetPopulatedTicksInWord = "getPopulatedTicksInWord"
	erc20MethodBalanceOf                  = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{Swap: 125000}
)
