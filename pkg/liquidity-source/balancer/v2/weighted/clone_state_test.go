package weighted

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCloneStateIsIndependent checks that UpdateBalance on a clone changes neither the original's reserves
// nor the totals. Both are slices UpdateBalance writes by index.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	orig := &PoolSimulator{
		totalAmountsIn: []*uint256.Int{uint256.NewInt(10), uint256.NewInt(20)},
		basePools:      map[string]shared.IBasePool{},
	}
	orig.Info = pool.PoolInfo{Address: "parent", Tokens: []string{"0xaaa", "0xbbb"},
		Reserves: []*big.Int{big.NewInt(1000), big.NewInt(2000)}}

	c := orig.CloneState().(*PoolSimulator)
	c.updateBalance("0xaaa", "0xbbb", big.NewInt(10), big.NewInt(20))

	require.Equal(t, big.NewInt(1000).Cmp(orig.Info.Reserves[0]), 0, "original reserve 0 changed")
	require.Equal(t, big.NewInt(2000).Cmp(orig.Info.Reserves[1]), 0, "original reserve 1 changed")
	require.Equal(t, uint64(10), orig.totalAmountsIn[0].Uint64(), "original total 0 changed")
	require.Equal(t, uint64(20), orig.totalAmountsIn[1].Uint64(), "original total 1 changed")
	require.Equal(t, big.NewInt(1010).Cmp(c.Info.Reserves[0]), 0, "clone reserve 0 not updated")
}

// stableBasePool embeds a stable pool, so it has no CloneState (nil from pool.Pool). Its UpdateBalance
// writes reserves directly, since weighted passes no SwapInfo and stable's own UpdateBalance is a no-op.
type stableBasePool struct {
	*stable.PoolSimulator
}

func (b *stableBasePool) UpdateBalance(params pool.UpdateBalanceParams) {
	b.Info.Reserves[0] = new(big.Int).Add(b.Info.Reserves[0], params.TokenAmountIn.Amount)
	b.Info.Reserves[1] = new(big.Int).Sub(b.Info.Reserves[1], params.TokenAmountOut.Amount)
}

// TestCloneStateStableBasePoolIsIndependent checks that a clone's UpdateBalance does not write into the
// original's stable base pool. Stable has no CloneState, so the clone must not share it.
func TestCloneStateStableBasePoolIsIndependent(t *testing.T) {
	t.Parallel()
	stablePool := &stableBasePool{&stable.PoolSimulator{Info: pool.PoolInfo{Address: "stable",
		Tokens: []string{"0xaaa", "0xbbb"}, Reserves: []*big.Int{big.NewInt(1000), big.NewInt(2000)}}}}
	orig := &PoolSimulator{
		basePools: map[string]shared.IBasePool{"stable": stablePool},
	}
	orig.Info = pool.PoolInfo{Address: "parent", Tokens: []string{"0xaaa", "0xbbb"},
		Reserves: []*big.Int{big.NewInt(1000), big.NewInt(2000)}}

	c := orig.CloneState()
	if c != nil {
		c.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: "0xaaa", Amount: big.NewInt(10)},
			TokenAmountOut: pool.TokenAmount{Token: "0xbbb", Amount: big.NewInt(20)},
			SwapInfo: shared.SwapInfo{Hops: []shared.Hop{{
				Pool: "stable", TokenIn: "0xaaa", TokenOut: "0xbbb",
				AmountIn: uint256.NewInt(10), AmountOut: uint256.NewInt(20),
			}}},
		})
	}

	require.Equal(t, big.NewInt(1000).Cmp(stablePool.Info.Reserves[0]), 0, "original stable reserve 0 changed")
	require.Equal(t, big.NewInt(2000).Cmp(stablePool.Info.Reserves[1]), 0, "original stable reserve 1 changed")
}
