package balancer

import (
	"math/big"
	"time"
)

type PoolType string
type DexType string

const (
	// DexTypeBalancer is used to detect all types of balancer pools
	DexTypeBalancer = "balancer"

	subgraphPoolTypeWeighted   PoolType = "Weighted"
	subgraphPoolTypeStable     PoolType = "Stable"
	subgraphPoolTypeMetaStable PoolType = "MetaStable"
	DexTypeBalancerWeighted    DexType  = "balancer-weighted"
	DexTypeBalancerStable      DexType  = "balancer-stable"
	DexTypeBalancerMetaStable  DexType  = "balancer-meta-stable"

	graphQLRequestTimeout = 20 * time.Second

	emptyString         = ""
	zeroString          = "0"
	zeroFloat64 float64 = 0

	// Contract methods

	vaultMethodGetPoolTokens = "getPoolTokens"

	// poolMethodGetVault to get vault of a pool
	poolMethodGetVault                    = "getVault"
	poolMethodGetSwapFeePercentage        = "getSwapFeePercentage"
	poolMethodGetAmplificationParameter   = "getAmplificationParameter"
	metaStablePoolMethodGetScalingFactors = "getScalingFactors"
)

var (
	// dexTypeByPoolType Add more types of pool here when we integrate a new type of Balancer
	dexTypeByPoolType = map[PoolType]DexType{
		subgraphPoolTypeWeighted:   DexTypeBalancerWeighted,
		subgraphPoolTypeStable:     DexTypeBalancerStable,
		subgraphPoolTypeMetaStable: DexTypeBalancerMetaStable,
	}

	zeroBI       = big.NewInt(0)
	bOneFloat, _ = new(big.Float).SetString("1000000000000000000")

	DefaultGas = Gas{Swap: 80000}
)
