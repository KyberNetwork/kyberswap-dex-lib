package clear

import (
	"math/big"
)

// Metadata for pagination in pool discovery
//
//	type Metadata struct {
//		LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
//	}
type Metadata struct {
	Offset map[string]int `json:"offset"`
}

// GraphQL response types
type (
	GraphQLToken struct {
		Address  string `json:"address"`
		Symbol   string `json:"symbol"`
		Decimals string `json:"decimals"`
	}

	GraphQLVault struct {
		ID      string         `json:"id"`
		Address string         `json:"address"`
		Tokens  []GraphQLToken `json:"tokens"`
	}

	GraphQLResponse struct {
		ClearVaults []GraphQLVault `json:"clearVaults"`
	}
)

// StaticExtra contains immutable pool data
type StaticExtra struct {
	SwapAddress string `json:"swapAddress"`
}

// Extra contains mutable pool state
type Extra struct {
	Reserves map[int]map[int]*PreviewSwapResult `json:"reserves"` // token address -> reserve
}

// PoolMeta contains metadata for swap execution
type PoolMeta struct {
	VaultAddress string `json:"vaultAddress"`
	SwapAddress  string `json:"swapAddress"`
}

// PreviewSwapResult from ClearSwap.previewSwap()
type PreviewSwapResult struct {
	AmountIn  *big.Int
	AmountOut *big.Int
	IOUs      *big.Int `json:"-"`
}

// Gas costs for different operations
type Gas struct {
	Swap int64
}

var DefaultGas = Gas{
	Swap: defaultGas,
}
