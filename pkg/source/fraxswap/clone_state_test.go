package fraxswap

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// CloneState gives a typed copy whose reserves a swap on the clone does not reach: UpdateBalance stores new
// reserve values into Info.Reserves, so the clone needs a slice of its own. Without CloneState, pathfinder-lib's
// DefaultClonePool falls back to go-clone's Slowly, which stores the clone's pointers without GC write barriers.
func TestCloneStateIsTypedAndIsolated(t *testing.T) {
	sim, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{"20000", "20000"},
		Tokens:   []*entity.PoolToken{{Address: "a"}, {Address: "b"}},
		Extra:    `{"reserve0": 20000, "reserve1": 20000, "fee": 9997}`,
	})
	require.NoError(t, err)

	cloned := sim.CloneState()
	require.NotNil(t, cloned, "CloneState must not leave the clone to go-clone")
	require.IsType(t, sim, cloned)
	testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(1000)},
		TokenOut:      "b",
	}, nil)
}
