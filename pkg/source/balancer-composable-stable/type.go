package balancercomposablestable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Metadata map[string]PoolTypeMetadata

type PoolTypeMetadata struct {
	LastCreateTime *big.Int `json:"lastCreateTime"`
}

type SubgraphPool struct {
	ID         string   `json:"id"`
	Address    string   `json:"address"`
	SwapFee    string   `json:"swapFee"`
	PoolType   string   `json:"poolType"`
	CreateTime *big.Int `json:"createTime"`
	Tokens     []struct {
		Address  string `json:"address"`
		Weight   string `json:"weight"`
		Decimals int    `json:"decimals"`
	} `json:"tokens"`
}

type StaticExtra struct {
	VaultAddress  string `json:"vaultAddress"`
	PoolId        string `json:"poolId"`
	TokenDecimals []int  `json:"tokenDecimals"`
}

type LastJoinExitData struct {
	LastJoinExitAmplification *big.Int
	LastPostJoinExitInvariant *big.Int
}

type TokenRateCache struct {
	Rate     *big.Int
	OldRate  *big.Int
	Duration *big.Int
	Expires  *big.Int
}

type Extra struct {
	AmplificationParameter              AmplificationParameter `json:"amplificationParameter"`
	ScalingFactors                      []*big.Int             `json:"scalingFactors,omitempty"`
	BptIndex                            *big.Int               `json:"bptIndex"`
	LastJoinExit                        *LastJoinExitData      `json:"lastJoinExit"`
	RateProviders                       []string               `json:"rateProviders"`
	TokensExemptFromYieldProtocolFee    []bool                 `json:"tokensExemptFromYieldProtocolFee"`
	TokenRateCaches                     []TokenRateCache       `json:"tokenRateCaches"`
	ProtocolFeePercentageCacheSwapType  *big.Int               `json:"protocolFeePercentageCacheSwapType"`
	ProtocolFeePercentageCacheYieldType *big.Int               `json:"protocolFeePercentageCacheYieldType"`
}

type PoolTokens struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type AmplificationParameter struct {
	Value      *big.Int `json:"value"`
	IsUpdating bool     `json:"isUpdating"`
	Precision  *big.Int `json:"precision"`
}

type Gas struct {
	Swap int64
}

type Meta struct {
	VaultAddress           string         `json:"vault"`
	PoolId                 string         `json:"poolId"`
	MapTokenAddressToIndex map[string]int `json:"mapTokenAddressToIndex"`
}
