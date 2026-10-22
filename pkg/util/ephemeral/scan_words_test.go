package ephemeral

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScanWordsReadsWindowsInParallel: the words a call leaves are split and read by calls at once,
// at the first call's block, and the items come back in word order as one scan would give them.
func TestScanWordsReadsWindowsInParallel(t *testing.T) {
	const perCall = 3 // words one call reads
	var inFlight, most atomic.Int32
	var blocks sync.Map
	fetch := func(_ context.Context, lower, upper int16, block *big.Int) (Page[int16], []int16, error) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
		}
		if block != nil {
			blocks.Store(block.Uint64(), true)
		}
		// Long enough for the windows' calls to overlap, as a node's round trips do.
		time.Sleep(5 * time.Millisecond)
		var words []int16
		w := lower
		for ; w <= upper && len(words) < perCall; w++ {
			words = append(words, w)
		}
		return Page[int16]{Block: 100, Next: w, Done: w > upper}, words, nil
	}

	block, words, err := ScanWords(t.Context(), -10, 30, Pinned{}, WordsParallel, fetch)
	require.NoError(t, err)
	assert.EqualValues(t, 100, block)
	require.Len(t, words, 41)
	for i, w := range words {
		assert.EqualValues(t, -10+i, w, "word order")
	}
	assert.Greater(t, most.Load(), int32(1), "windows read at once")
	assert.LessOrEqual(t, most.Load(), int32(WordsParallel))
	blocks.Range(func(k, _ any) bool {
		assert.EqualValues(t, 100, k, "every window at the first call's block")
		return true
	})

	_, _, err = ScanWords(t.Context(), 0, 5, Pinned{},
		WordsParallel, func(context.Context, int16, int16, *big.Int) (Page[int16], []int16, error) {
			return Page[int16]{Block: 100, Next: 0}, nil, nil
		})
	assert.ErrorIs(t, err, ErrNoProgress)

	_, _, err = ScanWords(t.Context(), 0, 5, Pinned{Block: big.NewInt(100)}, WordsParallel,
		func(context.Context, int16, int16, *big.Int) (Page[int16], []int16, error) {
			return Page[int16]{Block: 107, Done: true}, nil, nil
		})
	assert.ErrorIs(t, err, ErrUnpinned)
}
