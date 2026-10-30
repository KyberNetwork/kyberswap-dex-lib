package rateob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `mapstructure:"dexID" json:"dexID,omitempty"`
	ChainID valueobject.ChainID `mapstructure:"chainID" json:"chainID,omitempty"`
	// MatchingEngine: the swap entry point, the approval target, and the source of fees and limits.
	Engine string `mapstructure:"engine" json:"engine,omitempty"`
	// Executor is the account the engine pays (the `recipient` of limitBuy/limitSell). The taker
	// fee is looked up for this account, so it must be the address the executor passes.
	Executor string `mapstructure:"executor" json:"executor,omitempty"`
}

type Metadata struct {
	Offset int `json:"offset"`
}

// StaticExtra holds a pair's immutable conversion parameters. Orderbook.convert scales by
// DecDiff = 10^|baseDecimals - quoteDecimals|, dividing when BaseBQuote (base has more decimals).
type StaticExtra struct {
	Engine     string `json:"e"`
	DecDiff    uint64 `json:"d"`
	BaseBQuote bool   `json:"bq,omitempty"`
	// NativeOut is the engine's WETH when the pair holds it and the chain unwraps (nativeScale 0):
	// the book then pays a WETH leg as native coin.
	NativeOut string `json:"n,omitempty"`
}

// Order is one resting order at a level, in queue (FIFO) order. Deposit is base for an ask and
// quote for a bid; Deadline 0 is good-till-cancelled.
type Order struct {
	Deposit  uint256.Int `json:"d"`
	Deadline uint64      `json:"t,omitempty"`
}

// Level is one price level. More is set when the level holds orders beyond the ones read.
type Level struct {
	Price  uint256.Int `json:"p"`
	Orders []Order     `json:"o"`
	More   bool        `json:"m,omitempty"`
}

// Extra is empty (no levels) while the pair is not listed yet or has no last matched price.
type Extra struct {
	Lmp        uint256.Int `json:"l"`
	SpreadBuy  uint32      `json:"sb,omitempty"`
	SpreadSell uint32      `json:"ss,omitempty"`
	TakerFee   uint32      `json:"f,omitempty"`
	MaxMatches uint32      `json:"mm,omitempty"`
	Timestamp  uint64      `json:"ts,omitempty"`
	Bids       []Level     `json:"b,omitempty"`
	Asks       []Level     `json:"a,omitempty"`
}

// SwapInfo carries the order parameters the executor sends and what UpdateBalance consumes.
//
// The executor calls MatchingEngine.limitBuy (quote in) or limitSell (base in) with
// {base, quote, price: LimitPrice, amount, isMaker: false, n: N, recipient: executor}.
type SwapInfo struct {
	IsBuy      bool   `json:"isBuy"`
	LimitPrice string `json:"limitPrice"`
	N          uint32 `json:"n"`

	Levels  int          `json:"-"` // levels emptied, from the front of the consumed side
	Orders  int          `json:"-"` // orders removed from the front of the next level
	Head    *uint256.Int `json:"-"` // the next level's head order deposit after a partial fill, or nil
	Lmp     uint256.Int  `json:"-"` // last matched price after the sweep
	Matches int          `json:"-"`
}

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}

type rpcOrder struct {
	Owner         common.Address `abi:"owner"`
	Price         *big.Int       `abi:"price"`
	DepositAmount *big.Int       `abi:"depositAmount"`
	Deadline      uint64         `abi:"deadline"`
}
