package weighted

import (
	"errors"
)

const (
	DexType = "balancer-v3-weighted"

	SubgraphPoolType = "WEIGHTED"

	poolMethodGetNormalizedWeights = "getNormalizedWeights"
)

var (
	ErrInvalidToken = errors.New("invalid token")

	baseGas   int64 = 213745
	bufferGas int64 = 120534
)
