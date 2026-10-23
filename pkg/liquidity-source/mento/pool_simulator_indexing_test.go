package mento

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The oracle rate is valid at the snapshot block but long expired by wall-clock time. Indexing
// must evaluate the window at the snapshot; route finding must still reject it.
func TestIndexingQuotesAgedState(t *testing.T) {
	p := snapshotPool(t)
	quote := func(sim pool.IPoolSimulator) error {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(1_000_000)},
			TokenOut:      usdm,
		})
		return err
	}

	indexed, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	require.NoError(t, quote(indexed))

	routed, err := pool.Factory(DexType)(pool.FactoryParams{
		EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true},
	})
	require.NoError(t, err)
	require.ErrorIs(t, quote(routed), ErrNoRecentRate)
}
