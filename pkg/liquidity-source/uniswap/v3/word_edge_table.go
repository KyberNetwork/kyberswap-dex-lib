package uniswapv3

import (
	_ "embed"
	"sync/atomic"

	"github.com/holiman/uint256"
)

// Swaps toward lower ticks stop at a word's lowest tick, 256*word*tickSpacing: always a multiple of
// 256, so one table serves every spacing. downEdgeSqrtPrices holds GetSqrtRatioAtTick at each
// in-range multiple of 256, big-endian: 12 bytes below tick 0 (price < 2^96), 20 from tick 0 up.
// Regenerate with `go test -run TestDownEdgeSqrtPrices -update-down-edges`.
//
//go:embed down_edge_sqrt_prices.bin
var downEdgeSqrtPrices []byte

const (
	minDownEdge      = -(-MinTick >> 8) // lowest in-range tick/256
	maxDownEdge      = MaxTick >> 8
	negDownEdgeBytes = -minDownEdge * 12
	downEdgeBytes    = negDownEdgeBytes + (maxDownEdge+1)*20
)

// downEdgeFloats holds every down edge price split into floats, for walkWordsFloat.
var downEdgeFloats = func() []edgeF {
	floats := make([]edgeF, maxDownEdge-minDownEdge+1)
	for k := minDownEdge; k <= maxDownEdge; k++ {
		sqrtPrice := downEdgeSqrtPrice(k << 8)
		floats[k-minDownEdge] = splitFloat(&sqrtPrice)
	}
	return floats
}()

// downEdgeSqrtPrice returns GetSqrtRatioAtTick(tick) for an in-range multiple of 256.
func downEdgeSqrtPrice(tick int) (sqrtPrice uint256.Int) {
	if k := tick >> 8; k < 0 {
		i := (k - minDownEdge) * 12
		sqrtPrice.SetBytes12(downEdgeSqrtPrices[i : i+12])
	} else {
		i := negDownEdgeBytes + k*20
		sqrtPrice.SetBytes20(downEdgeSqrtPrices[i : i+20])
	}
	return sqrtPrice
}

// upEdgeTable holds GetSqrtRatioAtTick at every word's highest tick, (256*word+255)*tickSpacing,
// where swaps toward higher ticks stop. Those ticks differ per spacing, so each has its own table.
type upEdgeTable struct {
	minWord int
	prices  []uint256.Int
	floats  []edgeF // prices split into floats, for walkWordsFloat
}

// maxTickSpacing is Uniswap v4's int16 bound; larger spacings compute edge prices directly.
const maxTickSpacing = 1<<15 - 1

// upEdgeTables is indexed by tick spacing. Common spacings (Uniswap 1/10/60/200, Pancake 50,
// Slipstream 100/2000) are built at init, others on first use; a table never changes once stored.
var upEdgeTables [maxTickSpacing + 1]atomic.Pointer[upEdgeTable]

func init() {
	for _, tickSpacing := range []int{1, 10, 50, 60, 100, 200, 2000} {
		upEdgeTables[tickSpacing].Store(newUpEdgeTable(tickSpacing))
	}
}

// upEdgeTableFor returns the spacing's table, or nil above maxTickSpacing.
func upEdgeTableFor(tickSpacing int) *upEdgeTable {
	if tickSpacing > maxTickSpacing {
		return nil
	}
	slot := &upEdgeTables[tickSpacing]
	if t := slot.Load(); t != nil {
		return t
	}
	// Racing first callers may each build; the first store wins and the rest adopt it.
	if t := newUpEdgeTable(tickSpacing); slot.CompareAndSwap(nil, t) {
		return t
	}
	return slot.Load()
}

func newUpEdgeTable(tickSpacing int) *upEdgeTable {
	minWord, maxWord := floorDiv(MinTick, tickSpacing)>>8, floorDiv(MaxTick, tickSpacing)>>8
	t := &upEdgeTable{minWord: minWord, prices: make([]uint256.Int, maxWord-minWord+1),
		floats: make([]edgeF, maxWord-minWord+1)}
	for word := minWord; word <= maxWord; word++ {
		if tick := (word<<8 + 255) * tickSpacing; MinTick <= tick && tick <= MaxTick {
			_ = GetSqrtRatioAtTick(tick, &t.prices[word-minWord])
			t.floats[word-minWord] = splitFloat(&t.prices[word-minWord])
		}
	}
	return t
}

// sqrtPrice returns GetSqrtRatioAtTick(tick) for an in-range up edge. It returns by value so no
// caller can write into the shared table.
func (t *upEdgeTable) sqrtPrice(tick, tickSpacing int) uint256.Int {
	return t.prices[(tick/tickSpacing)>>8-t.minWord]
}
