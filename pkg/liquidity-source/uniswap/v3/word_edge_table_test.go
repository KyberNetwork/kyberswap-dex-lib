package uniswapv3

import (
	"flag"
	"os"
	"sync"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var updateDownEdges = flag.Bool("update-down-edges", false, "regenerate down_edge_sqrt_prices.bin")

// TestDownEdgeSqrtPrices checks the embedded file byte for byte against GetSqrtRatioAtTick, which is
// itself pinned to TickMath on every tick, and regenerates it under -update-down-edges.
func TestDownEdgeSqrtPrices(t *testing.T) {
	want := make([]byte, 0, downEdgeBytes)
	var sqrtPrice uint256.Int
	for k := minDownEdge; k <= maxDownEdge; k++ {
		require.NoError(t, GetSqrtRatioAtTick(k<<8, &sqrtPrice))
		b := sqrtPrice.Bytes20()
		if k < 0 {
			require.LessOrEqual(t, sqrtPrice.BitLen(), 96, "negative ticks must fit 12 bytes")
			want = append(want, b[8:]...)
		} else {
			want = append(want, b[:]...)
		}
	}
	if *updateDownEdges {
		require.NoError(t, os.WriteFile("down_edge_sqrt_prices.bin", want, 0o644))
		t.Skip("regenerated; rerun without -update-down-edges")
	}
	require.Equal(t, want, downEdgeSqrtPrices)
	for k := minDownEdge; k <= maxDownEdge; k++ {
		require.Equal(t, sqrtAt(t, k<<8), downEdgeSqrtPrice(k<<8), "tick %d", k<<8)
	}
}

// TestEdgeSqrtPrices checks every in-range word edge the swap loop can ask for, through the same
// lookups it uses, for precomputed, lazily built, misaligned (7) and extreme (maxTickSpacing)
// spacings.
func TestEdgeSqrtPrices(t *testing.T) {
	t.Parallel()

	for _, spacing := range []int{1, 7, 10, 50, 60, 100, 200, 2000, maxTickSpacing} {
		up := upEdgeTableFor(spacing)
		require.Same(t, up, upEdgeTableFor(spacing), "one table per spacing")
		checked := 0
		for word := floorDiv(MinTick, spacing) >> 8; word <= floorDiv(MaxTick, spacing)>>8; word++ {
			for _, zeroForOne := range []bool{true, false} {
				edge := wordBoundaryTick((word<<8+100)*spacing, spacing, zeroForOne)
				if edge < MinTick || edge > MaxTick {
					continue
				}
				var got uint256.Int
				if zeroForOne {
					got = downEdgeSqrtPrice(edge)
				} else {
					got = up.sqrtPrice(edge, spacing)
				}
				require.Equal(t, sqrtAt(t, edge), got, "spacing %d edge %d", spacing, edge)
				checked++
			}
		}
		require.Positive(t, checked)
	}
	require.Nil(t, upEdgeTableFor(maxTickSpacing+1))
}

// TestUpEdgeTableConcurrentFirstUse builds a spacing no other test touches from many goroutines at
// once: all must get the same table. Run with -race.
func TestUpEdgeTableConcurrentFirstUse(t *testing.T) {
	t.Parallel()

	const spacing = 12345
	tables := make([]*upEdgeTable, 32)
	var wg sync.WaitGroup
	for i := range tables {
		wg.Go(func() { tables[i] = upEdgeTableFor(spacing) })
	}
	wg.Wait()
	for _, table := range tables {
		require.Same(t, tables[0], table)
	}
}
