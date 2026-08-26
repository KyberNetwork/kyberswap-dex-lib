package prismprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Pair is one element of getSupportedPairs()'s (address,address)[] return.
type Pair struct {
	Token0 common.Address
	Token1 common.Address
}

// Order is one resting order in a getOrderBook() side: an (amountIn, amountOut)
// quote, not a cumulative tier -- see order-book package's Level type, which
// this gets converted into.
type Order struct {
	AmountIn  *big.Int
	AmountOut *big.Int
}

// Side is one of getOrderBook()'s two returned order lists. S1-S4 are present
// on-chain but unused here (see prism-prop discovery notes: no on-chain fee
// getter was found, and these scalars aren't needed to build the ladder).
type Side struct {
	Orders []Order
	S1     *big.Int
	S2     *big.Int
	S3     *big.Int
	S4     *big.Int
}

// OrderBook mirrors getOrderBook(address,address)'s single dynamic-tuple
// return value.
type OrderBook struct {
	Token0      common.Address
	Token1      common.Address
	BlockNumber *big.Int
	Side0       Side
	Side1       Side
}

// getOrderBookResult wraps OrderBook because go-ethereum's abi.Copy, when a
// method has exactly one return value, assigns the whole unpacked value into
// the FIRST FIELD of the destination struct (Arguments.copyAtomic) rather
// than into the destination struct itself. Passing *OrderBook directly here
// would silently assign the entire order book into OrderBook.Token0.
type getOrderBookResult struct {
	Book OrderBook
}

// StaticExtra is stored in entity.Pool.StaticExtra and never changes: the
// single router contract every prism-prop pool quotes and swaps through
// (there's one router per chain, unlike titan-prop's per-venue contracts).
type StaticExtra struct {
	RouterAddress string `json:"router"`
}

// PoolMeta is returned by GetMetaInfo so aggregator-encoding can call the
// router's own swap directly -- see PackPrismProp.
type PoolMeta struct {
	RouterAddress string `json:"router"`
	BlockNumber   uint64 `json:"blockNumber"`
}
