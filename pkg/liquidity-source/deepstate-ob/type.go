package deepstateob

import (
	"math/big"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
)

// StaticExtra is immutable per-pool metadata computed once at pool-list time.
type StaticExtra struct {
	// Router is the DeepstateV1 address that owns this book.
	Router string `json:"r"`
	// Lens is the deployed DeepstateBookLens address for Router, or empty to
	// force the naive per-node tree() BFS fallback (see pool_tracker.go).
	Lens string `json:"l,omitempty"`
	// Decimals mirrors entity.Pool.Tokens[i].Decimals for quick access
	// without re-reading the pool's token list.
	Decimals [2]uint8 `json:"d"`
}

// Extra wraps the generic order-book package's flat levels with the one
// extra field fill()/fillRoute() calldata needs that isn't part of that
// generic shape: the active book epoch.
type Extra struct {
	orderbook.Extra
	Epoch string `json:"e,omitempty"`
}

// MetaInfo is returned from GetMetaInfo for aggregator-encoding to build
// fill() calldata.
type MetaInfo struct {
	Router      string `json:"router,omitempty"`
	Epoch       string `json:"epoch,omitempty"`
	IsBid       bool   `json:"isBid,omitempty"`
	BlockNumber uint64 `json:"blockNumber,omitempty"`
}

// FeeConfigRPC mirrors DeepstateV1.feeConfig()'s two named outputs.
type FeeConfigRPC struct {
	Recipient string `abi:"recipient"`
	Bps       uint16 `abi:"bps"`
}

// RootsRPC mirrors DeepstateV1.roots()'s two named outputs.
type RootsRPC struct {
	AskRoot [32]byte `abi:"askRoot"`
	BidRoot [32]byte `abi:"bidRoot"`
}

// TreeRPC mirrors DeepstateV1.tree()'s two named outputs (a branch's child
// pointers, or (0,0) when the queried node is a leaf).
type TreeRPC struct {
	LeftNode  [32]byte `abi:"leftNode"`
	RightNode [32]byte `abi:"rightNode"`
}

// LensSideRPC mirrors DeepstateBookLens.Side.
type LensSideRPC struct {
	Ticks      []int32    `abi:"ticks"`
	Quantities []*big.Int `abi:"quantities"`
	Truncated  bool       `abi:"truncated"`
}

// LensBookRPC mirrors DeepstateBookLens.Book -- the single struct returned
// by getBook(token0,token1,maxNodes), collapsing feeConfig/poolEpoch/roots/
// both BFS walks into one eth_call.
type LensBookRPC struct {
	Epoch  *big.Int    `abi:"epoch"`
	FeeBps uint16      `abi:"feeBps"`
	Bid    LensSideRPC `abi:"bid"`
	Ask    LensSideRPC `abi:"ask"`
}
