package liquiditybookv20

import (
	"cmp"
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type binState struct {
	ID   uint32
	X, Y string
}

func snapshot(s *PoolSimulator) ([]string, []binState, uint32, feeParameters) {
	res := make([]string, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		res[i] = r.String()
	}
	bins := make([]binState, len(s.bins))
	for i, b := range s.bins {
		bins[i] = binState{b.ID, b.ReserveX.String(), b.ReserveY.String()}
	}
	return res, bins, s.activeBinID, s.feeParams
}

// CloneState gives a typed copy. Without one, pathfinder-lib's DefaultClonePool falls back to go-clone's Slowly,
// which stores the clone's pointers without GC write barriers: a lost pointer under a concurrent mark, and a bad
// pointer in the heap after it. Swaps on the clone leave the original as it
// was, and the clone quotes as a fresh simulator given the same swaps.
func TestCloneStateIsTypedAndIsolated(t *testing.T) {
	const active = 8388608
	e18 := bignumber.TenPowInt(18)
	var bins []Bin
	for i := uint32(1); i <= 6; i++ {
		bins = append(bins, Bin{ID: active - i, ReserveX: new(big.Int), ReserveY: new(big.Int).Set(e18)})
		bins = append(bins, Bin{ID: active + i, ReserveX: new(big.Int).Set(e18), ReserveY: new(big.Int)})
	}
	slices.SortFunc(bins, func(a, b Bin) int { return cmp.Compare(a.ID, b.ID) })
	orig := newTestSim(t, withActiveBin(bins, active), active)
	twin := newTestSim(t, withActiveBin(bins, active), active)

	cloned := orig.CloneState()
	require.NotNil(t, cloned, "CloneState must not leave the clone to go-clone")
	c := cloned.(*PoolSimulator)
	r0, b0, a0, f0 := snapshot(orig)

	amt := new(big.Int).Mul(big.NewInt(3), e18)
	for _, sim := range []*PoolSimulator{c, twin} {
		for _, in := range []string{"0xa", "0xb", "0xa"} {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: otherToken[in]})
			require.NoError(t, err)
			sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt},
				TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})
		}
	}
	r1, b1, a1, f1 := snapshot(orig)
	require.Equal(t, r0, r1, "the original's reserves moved")
	require.Equal(t, b0, b1, "the original's bins moved")
	require.Equal(t, a0, a1, "the original's active bin moved")
	require.Equal(t, f0, f1, "the original's fee parameters moved")
	rc, bc, ac, fc := snapshot(c)
	rt, bt, at, ft := snapshot(twin)
	require.Equal(t, rt, rc)
	require.Equal(t, bt, bc)
	require.Equal(t, at, ac)
	require.Equal(t, ft, fc)
	require.NotEqual(t, r0, rc, "the swaps must move the clone")
}
