package stablesfast

import "errors"

var (
	// ErrPoolIsNotTracked guards a quote taken before Track has read the pool's rate. Pricing
	// a zero would quote the whole skim away, so refuse rather than guess.
	ErrPoolIsNotTracked = errors.New("pool is not tracked")

	// ErrFeeAboveMax means the hook answered above its own bytecode ceiling, so the contract
	// at this address is not the one this package models.
	ErrFeeAboveMax = errors.New("fee pips above MAX_FEE_PIPS")
)
