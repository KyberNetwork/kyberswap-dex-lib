package ticks

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/KyberNetwork/logger"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// noFallback fails t if StateFromLens bootstraps without the lens.
func noFallback(t *testing.T) func(context.Context) (entity.Pool, error) {
	return func(context.Context) (entity.Pool, error) {
		t.Error("bootstrapped without the lens")
		return entity.Pool{}, errors.New("unexpected")
	}
}

// TestStateFromLensNeverReadsBelowWhatThePoolHasSeen: a batch whose head is older than the pool's
// own block or the logs it has seen without state, as a backfill chunk handled with a newer pool's
// task is, is not read at that head; the node's newest block is.
func TestStateFromLensNeverReadsBelowWhatThePoolHasSeen(t *testing.T) {
	stateless := entity.Pool{Address: "0xp", BlockNumber: 900, Extra: `{"lensDue":1000}`}
	var reads []*big.Int
	bootstrap := func(_ context.Context, block *big.Int) (entity.Pool, uint64, error) {
		reads = append(reads, block)
		read := uint64(1200)
		if block != nil {
			read = block.Uint64()
		}
		return entity.Pool{Address: "0xp", BlockNumber: read, Extra: `{"liquidity":"1"}`}, read, nil
	}
	headers := func(block uint64) pool.GetNewPoolStateParams {
		return pool.GetNewPoolStateParams{BlockHeaders: map[uint64]entity.BlockHeader{block: {}}}
	}

	got := StateFromLens(t.Context(), nil, stateless, headers(600), bootstrap, noFallback(t), logger.WithFields(nil))
	assert.Equal(t, []*big.Int{nil}, reads, "a head below lensDue is not read")
	assert.EqualValues(t, 1200, got.BlockNumber)

	reads = nil
	StateFromLens(t.Context(), nil, entity.Pool{Address: "0xp", BlockNumber: 900}, headers(800), bootstrap,
		noFallback(t), logger.WithFields(nil))
	assert.Equal(t, []*big.Int{nil}, reads, "a head below the pool's own block is not read")

	reads = nil
	got = StateFromLens(t.Context(), nil, stateless, headers(1100), bootstrap, noFallback(t), logger.WithFields(nil))
	assert.Len(t, reads, 1)
	assert.EqualValues(t, 1100, reads[0].Uint64(), "a head holding every log is read")
	assert.EqualValues(t, 1100, got.BlockNumber)
}

// TestStateFromLensKeepsTheLatestRead: a batch without headers, here one holding a reorg's removed
// log, is read at the node's newest block, and that read is the pool's state: its logs are all at
// or below it, and reading again what they touch at their own blocks would take it back.
func TestStateFromLensKeepsTheLatestRead(t *testing.T) {
	read := entity.Pool{Address: "0xp", BlockNumber: 1200, Extra: `{"liquidity":"1","ticks":[]}`}
	bootstrap := func(_ context.Context, block *big.Int) (entity.Pool, uint64, error) {
		assert.Nil(t, block, "a batch without headers is read at the latest")
		return read, 1200, nil
	}
	param := pool.GetNewPoolStateParams{Logs: []ethtypes.Log{
		{BlockNumber: 1100, Removed: true}, {BlockNumber: 1100},
	}}
	got := StateFromLens(t.Context(), nil, entity.Pool{Address: "0xp"}, param, bootstrap, noFallback(t),
		logger.WithFields(nil))
	assert.Equal(t, read, got)
}

// TestStateFromLensBootstrapsWithoutTheLensOnANodeThatCannotServeIt: a node that ignores a pinned
// block fails every lens read of a pool across blocks; the pool is bootstrapped without the lens
// rather than left without state, update after update. Any other failure leaves it for its next
// update, as does a bootstrap without the lens that fails.
func TestStateFromLensBootstrapsWithoutTheLensOnANodeThatCannotServeIt(t *testing.T) {
	stateless := entity.Pool{Address: "0xp", BlockNumber: 900}
	param := pool.GetNewPoolStateParams{Logs: []ethtypes.Log{{BlockNumber: 1000}}}
	failing := func(err error) LensBootstrap {
		return func(context.Context, *big.Int) (entity.Pool, uint64, error) { return entity.Pool{}, 0, err }
	}
	read := entity.Pool{Address: "0xp", BlockNumber: 1000, Extra: `{"liquidity":"1"}`}
	withoutLens := func(context.Context) (entity.Pool, error) { return read, nil }

	got := StateFromLens(t.Context(), nil, stateless, param, failing(ephemeral.ErrUnpinned), withoutLens,
		logger.WithFields(nil))
	assert.Equal(t, read, got)

	got = StateFromLens(t.Context(), nil, stateless, param, failing(errors.New("timeout")), noFallback(t),
		logger.WithFields(nil))
	assert.False(t, HasState(got), "any other failure leaves the pool without state")

	got = StateFromLens(t.Context(), nil, stateless, param, failing(ephemeral.ErrNoProgress),
		func(context.Context) (entity.Pool, error) { return entity.Pool{}, errors.New("subgraph down") },
		logger.WithFields(nil))
	assert.False(t, HasState(got), "a failed bootstrap without the lens leaves the pool without state")
	assert.Equal(t, WithLensDue(stateless, param.Logs), got)
}
