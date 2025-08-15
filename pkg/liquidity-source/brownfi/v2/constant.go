package brownfiv2

import (
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "brownfi-v2"

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodMinPriceAge    = "minPriceAge"
	factoryMethodPriceOf        = "priceOf"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
	pairMethodFee         = "fee"
	pairMethodKappa       = "k"
	pairMethodLambda      = "lambda"

	parsedDecimals = 18

	defaultGas = 183499
)

var (
	q64       = new(uint256.Int).Lsh(big256.U1, 64)
	q64x2     = new(uint256.Int).Mul(q64, big256.U2)
	q128      = big256.TwoPow128
	precision = big256.TenPow(8)

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidPrices            = errors.WithMessage(pool.ErrUnsupported, "invalid prices")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrMax80PercentOfReserve    = errors.New("MAX_80_PERCENT_OF_RESERVE")
)
