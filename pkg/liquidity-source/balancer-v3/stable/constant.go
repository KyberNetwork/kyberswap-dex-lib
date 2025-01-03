package stable

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"

const (
	DexType = "balancer-v3-stable"

	PoolType = "StablePool"

	poolMethodGetAmplificationParameter = "getAmplificationParameter"
)

var (
	defaultGas = shared.Gas{Swap: 80000}
)
