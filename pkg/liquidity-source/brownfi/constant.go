package brownfi

import (
	"errors"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "brownfi"

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"

	defaultGas = 150000
)

var (
	q128   = big256.U2Pow128
	q128x2 = new(uint256.Int).Mul(q128, big256.U2)

	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)
