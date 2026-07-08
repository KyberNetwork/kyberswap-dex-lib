package someswapv1

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "someswap-v1"

	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodGetPair        = "allPairs"
	pairMethodToken0            = "token0"
	pairMethodToken1            = "token1"
	pairMethodGetReserves       = "getReserves"
	pairMethodFee               = "fee"

	bps       = 10000
	maxFeeBps = bps - 1

	defaultGas = 111490
)

var (
	bpsDen    = big256.UBasisPoint
	weightDen = big256.BONE

	ErrInvalidTokenIndex = errors.New("invalid token index")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
	ErrInvalidReserves   = errors.New("invalid reserves")
	ErrAmountInTooSmall  = errors.New("amount in too small")
	ErrInvalidTotalOut   = errors.New("invalid total out")
	ErrInvalidNetOut     = errors.New("invalid net out")
)
