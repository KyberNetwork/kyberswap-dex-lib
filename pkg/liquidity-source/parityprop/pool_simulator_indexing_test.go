package parityprop

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The tracker stamps Pool.Timestamp with the state's block time, which can trail the indexer's
// wall clock. Indexing must quote as of that snapshot; route finding still rejects it.
func TestIndexingQuotesAgedState(t *testing.T) {
	ep := livePool(t) // Timestamp == oracleUpdatedAt, maxStaleness 8s
	quote := func(sim pool.IPoolSimulator) error {
		_, err := pool.DefaultCalcAmtOut(context.Background(), sim, pool.TokenAmount{Token: weth, Amount: big.NewInt(1e17)}, usdg, nil)
		return err
	}

	indexed, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	require.NoError(t, quote(indexed))

	routed, err := pool.Factory(DexType)(pool.FactoryParams{
		EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: true},
	})
	require.NoError(t, err)
	assert.ErrorIs(t, quote(routed), ErrStalePrice)
}
