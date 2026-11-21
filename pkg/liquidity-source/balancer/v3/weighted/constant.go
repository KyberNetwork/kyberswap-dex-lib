package weighted

import (
	"errors"
)

const (
	DexType = "balancer-v3-weighted"

	SubgraphPoolType = "WEIGHTED"

	PoolMethodGetNormalizedWeights = "getNormalizedWeights"
	PoolMethodGetMinTokenBalances  = "getMinTokenBalances"
	PoolMethodVersion              = "version"

	baseGas = 213745
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenBalanceBelowMin    = errors.New("token balance below min")
	ErrMissingPoolVersion      = errors.New("missing weighted pool version")
	ErrMissingMinTokenBalances = errors.New("missing min token balances for weighted pool v2")
)
