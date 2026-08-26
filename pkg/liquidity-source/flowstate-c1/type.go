package flowstatec1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// StaticExtra is set once at pool creation. Market and Pool are the real on-chain
// addresses; Pool is shared across every quote asset a token can be bought with, so
// each (Pool, QuoteAsset) pair gets its own synthetic entity.Pool keyed off both.
type StaticExtra struct {
	Market     string `json:"m"`
	Pool       string `json:"p"`
	QuoteAsset string `json:"q"`
}

// Extra is refreshed by the tracker from quoteBuyFromPool(Pool, QuoteAsset, ProbeAmount).
// Pricing has no curve impact up to FillableAmount (vendor spec 3.1), so ProbeQuoteCost /
// ProbeAmount is the flat unit rate for any amount up to that cap.
type Extra struct {
	Available      bool         `json:"a"`
	ProbeAmount    *uint256.Int `json:"pa"`
	ProbeQuoteCost *uint256.Int `json:"pq"`
	FillableAmount *uint256.Int `json:"f"`
	FeeBps         uint16       `json:"fb"`
}

// Quote mirrors the Market's on-chain Quote struct exactly, so its fields stay
// *big.Int -- that's the type go-ethereum's ABI decoder unpacks a uint256 into at
// the RPC boundary. Convert to uint256.Int only once decoded, in the tracker.
type Quote struct {
	Available      bool
	FillableAmount *big.Int
	QuoteAmount    *big.Int
	FeeAmount      *big.Int
	FeeBps         uint16
	QuoteAsset     common.Address
}

type MetaInfo struct {
	Market      string `json:"market"`
	Pool        string `json:"pool"`
	QuoteAsset  string `json:"quoteAsset"`
	BlockNumber uint64 `json:"blockNumber"`
}
