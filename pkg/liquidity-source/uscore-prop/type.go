package uscoreprop

import (
	"math/big"

	"github.com/holiman/uint256"
)

// Point is one on-chain quote of the pool: amountIn (index 0) returns amountOut (index 1).
type Point [2]uint256.Int

// Extra holds the sampled quote curve per direction.
type Extra struct {
	Ladders [2][]Point `json:"l"`
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
