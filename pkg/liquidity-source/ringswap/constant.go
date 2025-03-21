package ringswap

import (
	"errors"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
)

const (
	DexType     = "ringswap"
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

var (
	defaultGas = uniswapv2.Gas{Swap: 225000}
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
)

const (
	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"

	pairMethodBalanceOf = "balanceOf"

	fewWrappedTokenGetTokenMethod = "token"
)

var (
	ErrReserveIndexOutOfBounds = errors.New("reserve index out of bounds")
	ErrTokenIndexOutOfBounds   = errors.New("token index out of bounds")
	ErrTokenSwapNotAllowed     = errors.New("cannot swap between original token and wrapped token")

	ErrNoSwapLimit = errors.New("swap limit is required")
)
