package uscoreprop

import "github.com/holiman/uint256"

// Point is one on-chain quote of the pool: amountIn (index 0) returns amountOut (index 1).
type Point [2]*uint256.Int

// Extra holds the sampled quote curve per direction. Time is the snapshot block time: it changes
// on every refresh, so pool-service persists the new Timestamp even when the oracle (and so the
// ladder) hasn't moved; otherwise the unchanged-state skip would let the pool trip MaxAge.
type Extra struct {
	Ladders [2][]Point `json:"l"`
	Time    int64      `json:"t"`
}
