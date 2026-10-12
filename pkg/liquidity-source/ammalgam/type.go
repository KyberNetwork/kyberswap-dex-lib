package ammalgam

import "github.com/holiman/uint256"

// Extra is the mutable state captured by the tracker at one block.
type Extra struct {
	ReferenceReserves [2]uint256.Int `json:"referenceReserves"`
	MissingAssets     [2]uint256.Int `json:"missingAssets"`
}

// StaticExtra holds metadata that does not change with a quote.
type StaticExtra struct {
	FactoryAddress     string `json:"factoryAddress,omitempty"`
	PairBeacon         string `json:"pairBeacon,omitempty"`
	PairImplementation string `json:"pairImplementation,omitempty"`
	Gas                int64  `json:"gas,omitempty"`
}

// SwapInfo carries the full post-swap pair reserves, so UpdateBalance does not
// need to repeat the quote or alter the snapshot's missing/reference assets.
type SwapInfo struct {
	PoolAddress string
	PreReserves [2]uint256.Int
	Reserves    [2]uint256.Int
	InputIndex  int
	AmountIn    uint256.Int
	AmountOut   uint256.Int
}
