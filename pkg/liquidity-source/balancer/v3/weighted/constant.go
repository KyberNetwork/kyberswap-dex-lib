package weighted

import (
	"errors"
)

const (
	DexType = "balancer-v3-weighted"

	SubgraphPoolType = "WEIGHTED"

	PoolMethodGetNormalizedWeights = "getNormalizedWeights"
	PoolMethodGetMinTokenBalances  = "getMinTokenBalances"

	baseGas = 213745
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenBalanceBelowMin = errors.New("token balance below min")
)
