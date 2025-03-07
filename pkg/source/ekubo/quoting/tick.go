package quoting

import (
	"cmp"
	"math"
	"math/big"
	"slices"
)

const (
	InvalidTickIndex  int   = -1
	invalidTickNumber int32 = math.MinInt32
)

type Tick struct {
	Number         int32    `json:"number"`
	LiquidityDelta *big.Int `json:"liquidityDelta"`
}

func NearestInitializedTickIndex(sortedTicks []Tick, tickNumber int32) int {
	idx, found := slices.BinarySearchFunc(sortedTicks, tickNumber, func(tick Tick, tickNumber int32) int {
		return cmp.Compare(tick.Number, tickNumber)
	})

	if !found {
		if idx == 0 {
			idx = InvalidTickIndex
		} else {
			idx -= 1
		}
	}

	return idx
}
