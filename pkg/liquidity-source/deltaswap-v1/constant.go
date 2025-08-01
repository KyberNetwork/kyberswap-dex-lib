package deltaswapv1

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "deltaswap-v1"

	defaultGas = 225000
)

var (
	Number_20   = uint256.NewInt(20)
	Number_1000 = uint256.NewInt(1000)
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	factoryMethodDsFeeInfo                  = "dsFeeInfo"
	factoryMethodGetTradeLiquidityEMAParams = "getTradeLiquidityEMAParams"
	factoryMethodGetLiquidityEMA            = "getLiquidityEMA"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
)

var (
	ErrZeroTradeLiquidity = errors.New("DeltaSwap: ZERO_TRADE_LIQUIDITY")
	ErrMaxIterations      = errors.New("maximum iterations reached")
)
