package math

import "errors"

var (
	// ErrOverflow is Solidity's panic 0x11 and SafeCast's overflow revert: a checked operation, a
	// cast, or a rounded-up result that does not fit its type.
	ErrOverflow = errors.New("nightly-prop math: arithmetic overflow")
	// ErrDivisionByZero is Solidity's panic 0x12.
	ErrDivisionByZero = errors.New("nightly-prop math: division by zero")
	// ErrInexactAdjustment is PriceMathInexactAdjustment: an offset or spread adjustment would
	// round.
	ErrInexactAdjustment = errors.New("nightly-prop math: inexact price adjustment")
	// ErrDenominatorMismatch is PriceMathDenominatorMismatch: two prices with different
	// denominators were combined.
	ErrDenominatorMismatch = errors.New("nightly-prop math: price denominator mismatch")
	// ErrZeroSwapAmount is NightlyPoolZeroSwapAmount.
	ErrZeroSwapAmount = errors.New("nightly-prop math: zero swap amount")
	// ErrAmountTooSmall is NightlyPoolAmountTooSmall: the output rounds down to zero.
	ErrAmountTooSmall = errors.New("nightly-prop math: amount too small")
	// ErrInsufficientLiquidity is NightlyPoolInsufficientLiquidity: the walk passed the last level
	// with input left.
	ErrInsufficientLiquidity = errors.New("nightly-prop math: insufficient liquidity")
	// ErrLiquidityOverflow is NightlyPoolLiquidityOverflow: a level balance cannot record the
	// incoming amount.
	ErrLiquidityOverflow = errors.New("nightly-prop math: level liquidity overflow")
	// ErrStaleOraclePrice is NightlyPoolStaleOraclePrice: the price is older than the pool's
	// maximum price age at the block's time.
	ErrStaleOraclePrice = errors.New("nightly-prop math: stale oracle price")
	// ErrInvalidPriceLevel is NightlyPoolInvalidPriceLevel: a level ID outside the pool's levels.
	ErrInvalidPriceLevel = errors.New("nightly-prop math: invalid price level")
)
