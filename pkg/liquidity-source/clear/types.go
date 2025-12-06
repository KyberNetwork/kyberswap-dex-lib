package clear

import (
	"math/big"

	"github.com/holiman/uint256"
)

// Metadata for pagination in pool discovery
type Metadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
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
	VaultAddress string   `json:"vaultAddress"`
	SwapAddress  string   `json:"swapAddress"`
	Tokens       []string `json:"tokens"` // All token addresses in the vault
}

// Extra contains mutable pool state
type Extra struct {
	Reserves map[string]*uint256.Int `json:"reserves"` // token address -> reserve
	Paused   bool                    `json:"paused"`
}

// PoolMeta contains metadata for swap execution
type PoolMeta struct {
	VaultAddress string `json:"vaultAddress"`
	SwapAddress  string `json:"swapAddress"`
}

// PreviewSwapResult from ClearSwap.previewSwap()
type PreviewSwapResult struct {
	AmountOut *big.Int
	IOUs      *big.Int
}

// Gas costs for different operations
type Gas struct {
	Swap int64
}

var DefaultGas = Gas{
	Swap: defaultGas,
}
