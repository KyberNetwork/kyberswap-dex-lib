package prismprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Pair is one element of getSupportedPairs()'s return. The order inside a
// pair has no meaning.
type Pair struct {
	TokenA common.Address
	TokenB common.Address
}

// QuoteParams prices a book for the exact transaction described.
type QuoteParams struct {
	MsgSender common.Address
	TxOrigin  common.Address
	GasPrice  *big.Int
	PathType  uint8 // 0 = swap, 1 = swapWithCallback
}

// BookLevel is a marginal (not cumulative) level; both amounts are in native
// decimals, baseAmount in baseToken and quoteAmount in quoteToken.
type BookLevel struct {
	BaseAmount  *big.Int
	QuoteAmount *big.Int
}

// BookSide lists levels best price first, already cut to vault inventory and
// trade-size caps.
type BookSide struct {
	Levels             []BookLevel
	OutputVaultBalance *big.Int
	MinTradeSize       *big.Int
	MaxTradeSize       *big.Int
	Tif                uint64
}

// OrderBook mirrors getOrderBookWithParams' PairOrderBook. Bids: the taker
// sells base and receives quote. Asks: the taker buys base and pays quote.
type OrderBook struct {
	BaseToken     common.Address
	QuoteToken    common.Address
	SnapshotBlock uint64
	Bids          BookSide
	Asks          BookSide
}

// getOrderBookResult wraps OrderBook because go-ethereum's abi.Copy, when a
// method has exactly one return value, assigns the whole unpacked value into
// the FIRST FIELD of the destination struct (Arguments.copyAtomic) rather
// than into the destination struct itself.
type getOrderBookResult struct {
	Book OrderBook
}

// StaticExtra is stored in entity.Pool.StaticExtra and never changes: the
// single router contract every prism-prop pool quotes and swaps through
// (there's one router per chain).
type StaticExtra struct {
	RouterAddress string `json:"router"`
}
