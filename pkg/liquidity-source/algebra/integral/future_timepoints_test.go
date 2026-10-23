package integral

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReadAtAnOlderBlockDropsFutureTimepoints: a reset keeps the plugin's timepoints, cached at the
// latest block of the pool's last update. Read again at an older batch head, the slots past the head
// hold nothing that block had written, by lens and multicall alike.
func TestReadAtAnOlderBlockDropsFutureTimepoints(t *testing.T) {
	later := newFakeChain(1000, 300, false)
	later.write(7) // the last update read at the latest block, 7 timepoints on
	cached, _, _, _ := readBoth(t, later, nil, 0)

	head := newFakeChain(1000, 300, false) // the batch head the reset pool is read at
	got, _, _, _ := readBoth(t, head, cached.timepoints, 0)
	require.Zero(t, head.slots[1001], "slot 1001 is unwritten at the head")
	require.False(t, got.timepoints[1001].Initialized,
		"slot 1001 kept the cached ts %d, written after the block read", got.timepoints[1001].BlockTimestamp)
}

// TestFutureCacheMatchesTheChain: a cache read at a later state than the one read now, as a reset
// pool's read again at an older batch head is, leaves every slot read as the chain holds it, by lens
// and multicall alike.
func TestFutureCacheMatchesTheChain(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))
	for i := range 400 {
		index, cnt, full := uint16(rng.Uint32()), 1+rng.IntN(3000), rng.IntN(2) == 0
		spacing, age := 1+rng.Uint32N(1200), rng.Uint32N(2*WINDOW)
		perCall := []int{0, 0, 3, 100}[rng.IntN(4)]
		ahead := 1 + rng.IntN(400)
		t.Run(fmt.Sprintf("%d: index %d, %d by %ds, %ds old, full %v, %d a call, ahead %d", i, index, cnt, spacing,
			age, full, perCall, ahead), func(t *testing.T) {
			t.Parallel()
			mk := func() *fakeChain {
				c := newFakeChain(index, 0, full)
				for j := range cnt {
					c.slots[index-uint16(j)] = tpNow - age - uint32(j)*spacing
				}
				return c
			}
			later := mk()
			later.write(ahead)
			future, _, _, _ := readBoth(t, later, nil, perCall)
			earlier := mk()
			got, _, _, _ := readBoth(t, earlier, future.timepoints, perCall)
			if got.timepoints != nil {
				for idx, tp := range got.timepoints {
					if tp.Initialized && tp.BlockTimestamp != earlier.slots[idx] {
						t.Errorf("slot %d holds ts %d, the chain %d", idx, tp.BlockTimestamp, earlier.slots[idx])
					}
				}
			}
		})
	}
}
