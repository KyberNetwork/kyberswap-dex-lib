package composablestable

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func newCloneTestSimulator() *PoolSimulator {
	tokens := []string{"0xaaa", "0xbbb", "0xccc"}
	reserves := []*big.Int{big.NewInt(1000), big.NewInt(2000), big.NewInt(3000)}
	p := poolpkg.Pool{Info: poolpkg.PoolInfo{Address: "0xbpt", Tokens: tokens, Reserves: reserves}}
	cache := TokenRateCache{Rate: uint256.NewInt(5), OldRate: uint256.NewInt(7)}
	return &PoolSimulator{
		Pool:             p,
		regularSimulator: &regularSimulator{Pool: p},
		bptSimulator: &bptSimulator{
			Pool:            p,
			rateProviders:   []string{"0x1", "0x2", "0x3"},
			tokenRateCaches: []TokenRateCache{cache, cache, cache},
		},
	}
}

// TestCloneStateRegularSwapIsIndependent checks that a regular swap on a clone changes only the clone's
// reserves. Reserves are shared by all three pools in the original, so the clone must give them a fresh slice.
func TestCloneStateRegularSwapIsIndependent(t *testing.T) {
	t.Parallel()
	orig := newCloneTestSimulator()
	c := orig.CloneState().(*PoolSimulator)

	c.UpdateState(poolpkg.UpdateStateParams{
		TokenAmountIn:  poolpkg.TokenAmt{Token: "0xaaa", Amount: *uint256.NewInt(10)},
		TokenAmountOut: poolpkg.TokenAmt{Token: "0xbbb", Amount: *uint256.NewInt(20)},
	})

	require.Zero(t, orig.Info.Reserves[0].Cmp(big.NewInt(1000)), "original reserve 0 changed")
	require.Zero(t, orig.Info.Reserves[1].Cmp(big.NewInt(2000)), "original reserve 1 changed")
	require.Zero(t, c.Info.Reserves[0].Cmp(big.NewInt(1010)), "clone reserve 0 not updated")
	require.Zero(t, c.Info.Reserves[1].Cmp(big.NewInt(1980)), "clone reserve 1 not updated")
	require.Same(t, &c.Info.Reserves[0], &c.regularSimulator.Info.Reserves[0], "clone sub-simulator lost shared reserves")
}

// TestCloneStateBptSwapIsIndependent checks that a BPT swap on a clone leaves the original's reserves and
// tokenRateCaches unchanged. UpdateBalance writes both by index.
func TestCloneStateBptSwapIsIndependent(t *testing.T) {
	t.Parallel()
	orig := newCloneTestSimulator()
	c := orig.CloneState().(*PoolSimulator)

	c.UpdateState(poolpkg.UpdateStateParams{
		TokenAmountIn:  poolpkg.TokenAmt{Token: "0xbbb", Amount: *uint256.NewInt(10)},
		TokenAmountOut: poolpkg.TokenAmt{Token: "0xbpt", Amount: *uint256.NewInt(20)},
	})

	require.Zero(t, orig.Info.Reserves[1].Cmp(big.NewInt(2000)), "original reserve 1 changed")
	require.Equal(t, uint64(7), orig.bptSimulator.tokenRateCaches[1].OldRate.Uint64(), "original OldRate changed")
	require.Equal(t, uint64(5), c.bptSimulator.tokenRateCaches[1].OldRate.Uint64(), "clone OldRate not updated")
	require.Zero(t, c.Info.Reserves[1].Cmp(big.NewInt(2010)), "clone reserve 1 not updated")
}

// TestBasePoolAssertion checks the factory-built pool satisfies shared.IBasePool. Weighted and stable meta pools
// type-assert their base pools to shared.IBasePool, so a composable-stable base must pass that assertion.
func TestBasePoolAssertion(t *testing.T) {
	t.Parallel()
	var sim poolpkg.IPoolSim = newCloneTestSimulator()
	base, ok := sim.CloneState().(shared.IBasePool)
	require.True(t, ok, "composable-stable clone must satisfy shared.IBasePool")
	require.Equal(t, "0xbpt", base.GetAddress())
}
