package ticks

import (
	"errors"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestIsMissingTrieNodeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"missing trie node", errors.New("missing trie node abcd"), true},
		// This is the case that motivated delegating to the shared classifier: some
		// RPC providers report pruned state with this text instead of "missing trie
		// node", and the fallback-to-latest-block retry in pool_tracker.go's
		// updateState depends on this returning true for it too.
		{"historical state not available", errors.New("historical state 0xabc is not available"), true},
		{"unrelated error", errors.New("connection reset by peer"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsMissingTrieNodeError(tt.err))
		})
	}
}

// TestUnapplied: logs at or below the lens block are dropped, but never a removed log, whose undo
// the snapshot does not hold, nor the empty log that triggers an update.
func TestUnapplied(t *testing.T) {
	logs := []ethtypes.Log{{BlockNumber: 90}, {BlockNumber: 100}, {BlockNumber: 101}, {}}
	assert.Equal(t, logs, (&TicksBasedPool{}).Unapplied(logs))

	p := &TicksBasedPool{LensBlock: 100}
	assert.Equal(t, []ethtypes.Log{{BlockNumber: 101}, {}}, p.Unapplied(logs))
	assert.Equal(t, uint64(100), p.LensBlock)
	assert.Len(t, logs, 4, "the caller's slice is left as it was")
}

// TestUnappliedAfterAReorg: a reorg reverting block 95 under a snapshot at 100 means the snapshot
// holds the abandoned fork from 95 on, so the replacing block's logs must apply: dropping them as
// already held would leave the ticks they touch at the abandoned fork's values.
func TestUnappliedAfterAReorg(t *testing.T) {
	p := &TicksBasedPool{LensBlock: 100}
	logs := []ethtypes.Log{{BlockNumber: 90}, {BlockNumber: 95, Removed: true}, {BlockNumber: 95}, {BlockNumber: 96}}
	assert.Equal(t, []ethtypes.Log{{BlockNumber: 95, Removed: true}, {BlockNumber: 95}, {BlockNumber: 96}},
		p.Unapplied(logs))
	assert.Equal(t, uint64(94), p.LensBlock, "the snapshot holds the canonical chain up to 94 only")

	p = &TicksBasedPool{LensBlock: 100}
	p.Unapplied([]ethtypes.Log{{BlockNumber: 101, Removed: true}})
	assert.Equal(t, uint64(100), p.LensBlock, "a reorg above the snapshot leaves it whole")
}

// TestWithLensDue: the due block is the newest of the logs seen, never lowered, and the rest of the
// Extra is kept to the digit: a price is wider than a float64.
func TestWithLensDue(t *testing.T) {
	p := entity.Pool{Extra: `{"sqrtPriceX96":1461446703485210103287273052203988822378723970341,"tick":-5}`}
	p = WithLensDue(p, []ethtypes.Log{{BlockNumber: 90}, {}, {BlockNumber: 95, Removed: true}})
	assert.Equal(t, uint64(90), LensDue(p))
	assert.Contains(t, p.Extra, `"sqrtPriceX96":1461446703485210103287273052203988822378723970341`)
	assert.Contains(t, p.Extra, `"tick":-5`)

	p = WithLensDue(p, []ethtypes.Log{{BlockNumber: 80}})
	assert.Equal(t, uint64(90), LensDue(p), "an older log does not lower it")
	assert.Zero(t, LensDue(entity.Pool{}))
}

// TestWithoutState: a reset pool loses what its logs built and keeps the rest to the digit, and
// its lens snapshot must reach the block given or the one recorded already, whichever is higher.
func TestWithoutState(t *testing.T) {
	p := entity.Pool{Extra: `{"liquidity":7,"ticks":[{"index":1}],"lensBlock":50,"lensDue":120,` +
		`"sqrtPriceX96":1461446703485210103287273052203988822378723970341,"tick":-5}`}

	got := WithoutState(p, 100)
	assert.NotContains(t, got.Extra, `"liquidity"`)
	assert.NotContains(t, got.Extra, `"ticks"`)
	assert.NotContains(t, got.Extra, `"lensBlock"`)
	assert.Contains(t, got.Extra, `"sqrtPriceX96":1461446703485210103287273052203988822378723970341`)
	assert.Equal(t, uint64(120), LensDue(got), "an older block does not lower it")
	assert.Equal(t, uint64(130), LensDue(WithoutState(p, 130)))
}

// TestReadBlock: a batch's ticks are read at its newest block, never below the pool's, except in
// a reorg, which reads at its own block.
func TestReadBlock(t *testing.T) {
	assert.EqualValues(t, 0, ReadBlock(nil, 200))
	assert.EqualValues(t, 250, ReadBlock([]ethtypes.Log{{BlockNumber: 240}, {BlockNumber: 250}}, 200))
	assert.EqualValues(t, 200, ReadBlock([]ethtypes.Log{{BlockNumber: 150}}, 200), "a late log")
	assert.EqualValues(t, 150, ReadBlock([]ethtypes.Log{{BlockNumber: 150, Removed: true}, {BlockNumber: 150}}, 200),
		"a reorg")
}
