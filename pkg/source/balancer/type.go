package balancer

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

type Extra struct {
	AmplificationParameter AmplificationParameter `json:"amplificationParameter"`
	ScalingFactors         []*big.Int             `json:"scalingFactors,omitempty"`
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
