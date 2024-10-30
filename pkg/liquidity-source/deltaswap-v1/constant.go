package deltaswapv1

import (
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/holiman/uint256"
)

const (
	DexType = "deltaswap-v1"
)

var (
	defaultGas = uniswapv2.Gas{Swap: 225000}

	Number_20   = uint256.NewInt(20)
	Number_100  = uint256.NewInt(100)
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
