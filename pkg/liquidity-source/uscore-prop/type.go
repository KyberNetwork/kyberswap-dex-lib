package uscoreprop

import (
	"math/big"

	"github.com/holiman/uint256"
)

// Point is one on-chain quote of the pool: amountIn (index 0) returns amountOut (index 1).
type Point [2]uint256.Int

// Extra holds the sampled quote curve per direction. Time is the snapshot block time: it changes
// on every refresh, so pool-service persists the new Timestamp even when the oracle (and so the
// ladder) hasn't moved; otherwise the unchanged-state skip would let the pool trip MaxAge.
type Extra struct {
	Ladders [2][]Point `json:"l"`
	Time    int64      `json:"t"`
}

// ABI decode targets; amounts are converted to uint256 right after decoding.
type reservesResult struct{ ReserveQuote, ReserveBase *big.Int }
type quoteResult struct {
	AmountOut, Fee *big.Int
	Status         uint8
}
type ladderResult struct {
	Outs, Fees []*big.Int
	Status     uint8
}

// ladderQuote is a validated quoteLadder response: one output per sampled input.
type ladderQuote struct {
	outs   []*uint256.Int
	status uint8
}
