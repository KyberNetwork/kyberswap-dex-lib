package weighted

import (
	"errors"
)

const (
	DexType = "balancer-v3-weighted"

	SubgraphPoolType = "WEIGHTED"

	poolMethodGetNormalizedWeights = "getNormalizedWeights"

	baseGas = 213745
)

var (
	ErrInvalidToken = errors.New("invalid token")
)
