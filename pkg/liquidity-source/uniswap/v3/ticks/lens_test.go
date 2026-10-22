package ticks

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

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
	noUpdate := func(_ context.Context, p entity.Pool) (entity.Pool, error) { return p, nil }
	headers := func(block uint64) pool.GetNewPoolStateParams {
		return pool.GetNewPoolStateParams{BlockHeaders: map[uint64]entity.BlockHeader{block: {}}}
	}

	got := StateFromLens(context.Background(), nil, stateless, headers(600), bootstrap, noUpdate, logger.WithFields(nil))
	assert.Equal(t, []*big.Int{nil}, reads, "a head below lensDue is not read")
	assert.EqualValues(t, 1200, got.BlockNumber)

	reads = nil
	StateFromLens(context.Background(), nil, entity.Pool{Address: "0xp", BlockNumber: 900}, headers(800), bootstrap,
		noUpdate, logger.WithFields(nil))
	assert.Equal(t, []*big.Int{nil}, reads, "a head below the pool's own block is not read")

	reads = nil
	got = StateFromLens(context.Background(), nil, stateless, headers(1100), bootstrap, noUpdate, logger.WithFields(nil))
	assert.Len(t, reads, 1)
	assert.EqualValues(t, 1100, reads[0].Uint64(), "a head holding every log is read")
	assert.EqualValues(t, 1100, got.BlockNumber)
}
