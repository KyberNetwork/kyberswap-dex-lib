package everlongcvamm

import (
	"errors"
)

const (
	DexType = "everlong-cvamm"

	almMethodToken0             = "token0"
	almMethodToken1             = "token1"
	almMethodGetSupport         = "getSupport"
	almMethodXWad               = "xWad"
	almMethodAnchorSqrtCurveX96 = "anchorSqrtCurveX96"
	almMethodKappa              = "kappa"
	almMethodReserveStable      = "reserveStable"
	almMethodReserveVolatile    = "reserveVolatile"
	almMethodPoolFeeDirectional = "poolFeeDirectional"
	almMethodPaused             = "paused"
)

// Default gas per direction, overridable per deployment via Config/StaticExtra.
// The two legs differ structurally: volatile-in is closed form (one curve solve),
// stable-in runs a seeded bisection (~17 curve solves under DELEGATECALL). Totals
// include the two ERC20 transfers, the fee-hook read and reserve/coordinate writes;
// estimates pending a live deployment (gas is size-independent in both directions).
const (
	defaultGasStableIn   int64 = 160000
	defaultGasVolatileIn int64 = 110000
)

var (
	ErrExactOutNotSupported = errors.New("exact-output swaps are not supported (the fee is an output haircut; the curve solves forward only)")
	ErrInvalidToken         = errors.New("invalid token")
	ErrPaused               = errors.New("venue is paused")
	ErrRetractedBook        = errors.New("book is retracted (kappa == 0) or has no anchor")
	ErrCurveDomain          = errors.New("coordinate outside the curve domain")
	ErrCurveAmplification   = errors.New("amplification outside (WAD/2, 1000*WAD]")
	ErrSwapExhausted        = errors.New("nothing fills (input below normalized resolution or support exhausted)")
	ErrZeroAmountOut        = errors.New("zero amount out")
	ErrOverflow             = errors.New("amount overflow")
)
