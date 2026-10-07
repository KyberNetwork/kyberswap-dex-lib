package lglclob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

type OrderBookLevels struct {
	ArrayPrices []*uint256.Int `json:"p"`
	ArrayShares []*uint256.Int `json:"s"`
}

type OrderBook struct {
	Bids OrderBookLevels `json:"b"`
	Asks OrderBookLevels `json:"a"`
}

// Extra is a pool's state, its order book and what its market maker's quote
// costs.
type Extra struct {
	OrderBook
	// MakerQuoteGas is nil for a pool without a market maker, or whose market
	// maker was never measured.
	MakerQuoteGas *MakerQuoteGas `json:"g,omitempty"`
	// MeasuredAt is when MakerQuoteGas was last measured, in unix seconds.
	MeasuredAt int64 `json:"t,omitempty"`
}

// MakerQuoteGas is the gas of the quote a pool's market maker gives for each
// order, pricing the levels it posts for the order to fill, for a sell filling
// the bids and a buy filling the asks. A side is nil when it was not measured.
type MakerQuoteGas struct {
	Bids *LevelGas `json:"b,omitempty"`
	Asks *LevelGas `json:"a,omitempty"`
}

// LevelGas is the gas of a market maker's quote filling one level, and what
// each level more adds.
type LevelGas struct {
	First int64 `json:"f"`
	Next  int64 `json:"n"`
}

type StaticExtra struct {
	ScalingFactorX    *uint256.Int `json:"sX"`
	ScalingFactorY    *uint256.Int `json:"sY"`
	SupportsNativeEth bool         `json:"n,omitempty"`
}

type LobConfig struct {
	ScalingFactorTokenX           *big.Int
	ScalingFactorTokenY           *big.Int
	TokenX                        common.Address
	TokenY                        common.Address
	SupportsNativeEth             bool
	IsTokenXWeth                  bool
	AskTrie                       common.Address
	BidTrie                       common.Address
	AdminCommissionRate           uint64
	TotalAggressiveCommissionRate uint64
	TotalPassiveCommissionRate    uint64
	PassiveOrderPayoutRate        uint64
	ShouldInvokeOnTrade           bool
}

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type TokenInfo struct {
	ContractAddress string `json:"contractAddress"`
	Decimals        uint8  `json:"decimals"`
	Symbol          string `json:"symbol"`
	IsNative        bool   `json:"isNative"`
}

type MarketInfo struct {
	OrderbookAddress string      `json:"orderbookAddress"`
	BaseToken        TokenInfo   `json:"baseToken"`
	QuoteToken       TokenInfo   `json:"quoteToken"`
	AggressiveFee    json.Number `json:"aggressiveFee"`
}

type SwapInfo struct {
	executedLevels     int
	lastExecutedShares *uint256.Int
	HasNative          bool         `json:"e,omitempty"`
	PriceLimit         *uint256.Int `json:"p,omitempty"`
}
