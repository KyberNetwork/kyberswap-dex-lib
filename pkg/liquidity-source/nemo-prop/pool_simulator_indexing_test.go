package nemoprop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The tracker's row can trail the indexer's wall clock. Indexing must age the snapshot from
// Pool.Timestamp, not from now; route finding still refuses the same snapshot as stale.
func TestIndexingQuotesAgedState(t *testing.T) {
	const age = 120 * time.Second
	p := pooledState(t, linearLadders(2, 0.5), reserves1e9, age, 1000, 60_000, 10)
	p.Timestamp = time.Now().Add(-age).Unix()

	indexed, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	assert.Positive(t, quote1000(indexed))

	_, err = pool.Factory(DexType)(pool.FactoryParams{
		EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true},
	})
	assert.ErrorIs(t, err, ladder.ErrStale)
}
