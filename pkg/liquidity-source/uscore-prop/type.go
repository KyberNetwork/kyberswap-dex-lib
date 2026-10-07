package uscoreprop

import (
	"github.com/holiman/uint256"
)

// Point is one on-chain quote of the pool: amountIn (index 0) returns amountOut (index 1).
type Point [2]uint256.Int

// Extra holds the sampled quote curve per direction.
type Extra struct {
	Ladders [2][]Point `json:"l"`
}

// ladderQuote is a validated quoteLadder response: one output per sampled input.
type ladderQuote struct {
	outs   []*uint256.Int
	status uint8
}
