package umbraedamm

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeUmbraeDamm

	// defaultGas is a conservative estimate for a single DAMM pair.swap() call.
	defaultGas int64 = 125000

	// feeDenominator mirrors the contract's FEE_DENOMINATOR (basis points).
	feeDenominator = 10000

	pairMethodTokenX        = "tokenX"
	pairMethodTokenY        = "tokenY"
	pairMethodGetReserves   = "getReserves"
	pairMethodCurrentFeeBps = "currentFeeBps"
	pairMethodFeeToken      = "feeToken"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidReserve        = errors.New("invalid reserve")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInsufficientInput     = errors.New("insufficient input amount")
	ErrInsufficientOutput    = errors.New("insufficient output amount")
)
