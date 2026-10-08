package navjit

import "errors"

var (
	// ErrPoolIsNotTracked guards a quote taken before Track has read a ladder. The pool's own
	// tick math says nothing about a NavJit fill, so there is no safe default to fall back to.
	ErrPoolIsNotTracked = errors.New("navjit: pool is not tracked")

	// ErrNotAVenue: the pool is keyed at the hook but the hook does not list it as the venue of
	// either of its currencies, so the hook's swap path would revert.
	ErrNotAVenue = errors.New("navjit: pool is not an open NavJit venue")

	// ErrEmptyLadder: every V4Quoter rung in this direction reverted (NAV guard tripped, a
	// constituent pool dry, ...). The pool does not quote in that direction until the next Track.
	ErrEmptyLadder = errors.New("navjit: no quotable rung in this direction")

	// ErrBeyondLadder: the amount is above the largest size the V4Quoter answered for. The fill
	// curve beyond it is unknown (the hook may revert OutsideBand or SellExceedsFloat), so refuse.
	ErrBeyondLadder = errors.New("navjit: amount above the top quoted rung")

	// ErrSellExceedsFloat mirrors NavJitHook.SellExceedsFloat: a LOT-in swap can only redeem LOT
	// the PoolManager already holds.
	ErrSellExceedsFloat = errors.New("navjit: sell exceeds the PoolManager LOT float")

	// ErrInvalidAmount: a non-positive or over-256-bit specified amount.
	ErrInvalidAmount = errors.New("navjit: invalid amount")

	// ErrStale: under StaleCheck, the ladder is older than maxAgeSec.
	ErrStale = errors.New("navjit: ladder is stale")

	// ErrZeroOutput: the interpolated output rounds to zero (the hook would revert TradeTooSmall).
	ErrZeroOutput = errors.New("navjit: zero output")
)
