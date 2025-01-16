package weighted

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"

const (
	DexType = "balancer-v3-weighted"

	PoolType = "WeightedPool"

	poolMethodGetNormalizedWeights = "getNormalizedWeights"
)

var (
	defaultGas = shared.Gas{Swap: 80000}
)
