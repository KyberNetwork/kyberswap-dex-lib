package uniswapv3

import (
	"errors"
	"math/big"
)

const (
	DexTypeUniswapV3     = "uniswapv3"
	graphSkipLimit       = 5000
	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	defaultTokenWeight   = 50
	zeroString           = "0"
	emptyString          = ""
	rpcChunkSize         = 100
)

const (
	methodGetLiquidity                    = "liquidity"
	methodGetSlot0                        = "slot0"
	methodTickSpacing                     = "tickSpacing"
	tickLensMethodGetPopulatedTicksInWord = "getPopulatedTicksInWord"
	erc20MethodBalanceOf                  = "balanceOf"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 85000, CrossInitTickGas: 24000}

	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
	ErrTickNil        = errors.New("tick is nil")
	ErrV3TicksEmpty   = errors.New("v3Ticks empty")
)
