package fewshell

import "errors"

var (
	// ErrPoolIsNotTracked guards a quote taken before Track has read the route: a pool whose hX is empty or in
	// another hook's format has no pair, so there is nothing to price.
	ErrPoolIsNotTracked = errors.New("pool is not tracked")

	// ErrPaused is FewV2ShellHook's whenNotPaused revert.
	ErrPaused = errors.New("hook is paused")

	// ErrInvalidAmount is InvalidSwapAmount (a zero amount), or an amount past int128 (SafeCast.toInt128 reverts).
	ErrInvalidAmount = errors.New("invalid swap amount")

	// ErrInsufficientInventory is InsufficientPoolManagerInventory: the PoolManager holds less of the input token
	// than the hook must take().
	ErrInsufficientInventory = errors.New("insufficient pool manager inventory")

	// ErrInsufficientBacking is InsufficientWrapperBacking: the output Few token's contract holds less of its
	// underlying than the swap unwraps.
	ErrInsufficientBacking = errors.New("insufficient wrapper backing")

	// ErrDegeneratePair is DegeneratePair: a zero reserve on the FewV2 pair.
	ErrDegeneratePair = errors.New("degenerate few v2 pair")

	// ErrInsufficientAmount is FewV2Math's InsufficientAmount: the pair gives nothing, or the hook's net output is 0.
	ErrInsufficientAmount = errors.New("insufficient amount")

	// ErrInsufficientLiquidity is FewV2Math's InsufficientLiquidity: the output reaches the pair's reserve.
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")

	// ErrReserveOverflow is the pair's own uint112 balance check.
	ErrReserveOverflow = errors.New("few v2 reserve overflow")

	// ErrPairMismatch means the pair's tokens are not the pool's Few tokens, so the contract at this address is not
	// the one this package models.
	ErrPairMismatch = errors.New("few v2 pair does not match the route")

	// ErrBadBalance is a balance the chain returned that is missing or does not fit a uint256.
	ErrBadBalance = errors.New("bad balance read")
)
