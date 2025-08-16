package quantamm

import (
	"errors"
)

const (
	DexType = "balancer-v3-quantamm"

	SubgraphPoolType = "QUANT_AMM_WEIGHTED"

	poolMethodGetQuantAMMWeightedPoolDynamicData   = "getQuantAMMWeightedPoolDynamicData"
	poolMethodGetQuantAMMWeightedPoolImmutableData = "getQuantAMMWeightedPoolImmutableData"

	baseGas = 500000
)

var (
	ErrMaxTradeSizeRatioExceeded = errors.New("max trade size ratio exceeded")
)
