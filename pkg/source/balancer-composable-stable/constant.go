package balancercomposablestable

import (
	"math/big"
	"time"
)

type PoolType string
type DexType string

const (
	// DexTypeBalancerComposableStableExchange is used to detect all types of balancer pools
	DexTypeBalancerComposableStableExchange = "balancer-composable-stable"

	subgraphPoolTypeComposableStable PoolType = "ComposableStable"
	DexTypeBalancerComposableStable  DexType  = "balancer-composable-stable"
	graphQLRequestTimeout                     = 20 * time.Second

	emptyString         = ""
	zeroString          = "0"
	zeroFloat64 float64 = 0

	// Contract methods

	vaultMethodGetPoolTokens = "getPoolTokens"

	// poolMethodGetVault to get vault of a pool
	poolMethodGetVault                                          = "getVault"
	poolMethodGetSwapFeePercentage                              = "getSwapFeePercentage"
	poolMethodGetAmplificationParameter                         = "getAmplificationParameter"
	metaStablePoolMethodGetScalingFactors                       = "getScalingFactors"
	composableStablePoolMethodGetBptIndex                       = "getBptIndex"
	composableStablePoolMethodGetLastJoinExitData               = "getLastJoinExitData"
	composableStablePoolMethodGetTotalSupply                    = "totalSupply"
	composableStablePoolMethodIsTokenExemptFromYieldProtocolFee = "isTokenExemptFromYieldProtocolFee"
	composableStablePoolMethodGetRateProviders                  = "getRateProviders"
	composableStablePoolMethodGetTokenRateCache                 = "getTokenRateCache"
	composableStablePoolMethodGetProtocolFeePercentageCache     = "getProtocolFeePercentageCache"
)

var (
	// dexTypeByPoolType Add more types of pool here when we integrate a new type of Balancer
	dexTypeByPoolType = map[PoolType]DexType{
		subgraphPoolTypeComposableStable: DexTypeBalancerComposableStable,
	}

	zeroBI       = big.NewInt(0)
	bOneFloat, _ = new(big.Float).SetString("1000000000000000000")

	DefaultGas = Gas{Swap: 80000}

	/*
	   uint256 internal constant SWAP = 0;
	   uint256 internal constant FLASH_LOAN = 1;
	   uint256 internal constant YIELD = 2;
	   uint256 internal constant AUM = 3;
	*/
	ProtocolFeeTypeSwap  = big.NewInt(0)
	ProtocolFeeTypeYield = big.NewInt(2)
)
