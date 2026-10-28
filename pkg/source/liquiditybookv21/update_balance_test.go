package liquiditybookv21

import (
	"cmp"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// otherToken is the token a swap of 0xa or 0xb in a pair from newTestSim gives out.
var otherToken = map[string]string{"0xa": "0xb", "0xb": "0xa"}

func newTestSim(t *testing.T, bins []Bin, active uint32) *PoolSimulator {
	extra, err := json.Marshal(Extra{
		StaticFeeParams: staticFeeParams{BaseFactor: 10000, FilterPeriod: 30, DecayPeriod: 600,
			ReductionFactor: 5000, VariableFeeControl: 20000, ProtocolShare: 1000, MaxVolatilityAccumulator: 350000},
		VariableFeeParams: variableFeeParams{IdReference: 8388608, TimeOfLastUpdate: 1713521596},
		ActiveBinID:       active,
		BinStep:           20,
		Bins:              bins,
	})
	require.NoError(t, err)
	rx, ry := new(big.Int), new(big.Int)
	for _, b := range bins {
		rx.Add(rx, b.ReserveX)
		ry.Add(ry, b.ReserveY)
	}
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111",
		Exchange: "traderjoe-v21",
		Type:     DexTypeLiquidityBookV21,
		Reserves: entity.PoolReserves{rx.String(), ry.String()},
		Tokens:   []*entity.PoolToken{{Address: "0xa", Swappable: true}, {Address: "0xb", Swappable: true}},
		Extra:    string(extra),
	})
	require.NoError(t, err)
	return sim
}

// TestUpdateBalanceKeepsActiveBin: whatever swaps the simulator applies, starting from an empty
// active bin, the active bin stays in the set and no other empty bin does.
func TestUpdateBalanceKeepsActiveBin(t *testing.T) {
	const active = 8388608
	e18 := bignumber.TenPowInt(18)
	var bins []Bin
	for i := uint32(1); i <= 6; i++ {
		bins = append(bins, Bin{ID: active - i, ReserveX: new(big.Int), ReserveY: e18})
		bins = append(bins, Bin{ID: active + i, ReserveX: e18, ReserveY: new(big.Int)})
	}
	slices.SortFunc(bins, func(a, b Bin) int { return cmp.Compare(a.ID, b.ID) })
	sim := newTestSim(t, withActiveBin(bins, active), active)

	rng := rand.New(rand.NewPCG(1, 2))
	stats := map[string]int{}
	for step := range 2000 {
		in := []string{"0xa", "0xb"}[rng.IntN(2)]
		amt := new(big.Int).Mul(big.NewInt(rng.Int64N(9)+1), bignumber.TenPowInt(rng.Int64N(19)))
		var out *pool.TokenAmount
		var info any
		if rng.IntN(3) == 0 {
			res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: otherToken[in], Amount: amt}, TokenIn: in})
			if err != nil {
				stats["calcIn error"]++
				continue
			}
			out, info = &pool.TokenAmount{Token: otherToken[in], Amount: amt}, res.SwapInfo
			amt = res.TokenAmountIn.Amount
			stats["calcIn"]++
		} else {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: otherToken[in]})
			if err != nil {
				stats["calcOut error"]++
				continue
			}
			out, info = res.TokenAmountOut, res.SwapInfo
			stats["calcOut"]++
		}
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt},
			TokenAmountOut: *out, SwapInfo: info})

		_, err := sim.findBinArrIndex(sim.activeBinID)
		require.NoError(t, err, "step %d: active bin %d not in the set after a swap of %s %s", step, sim.activeBinID, amt, in)
		require.True(t, slices.IsSortedFunc(sim.bins, func(a, b BinU256) int { return cmp.Compare(a.ID, b.ID) }))
		for _, b := range sim.bins {
			if b.ID != sim.activeBinID {
				require.False(t, b.isEmpty(), "step %d: empty bin %d kept", step, b.ID)
			}
		}
	}
	require.Positive(t, stats["calcOut"], "swaps applied: %v", stats)
}

// TestUpdateBalanceZeroAmount: a swap of nothing leaves an empty active bin where it is.
func TestUpdateBalanceZeroAmount(t *testing.T) {
	const active = 8388608
	e18 := bignumber.TenPowInt(18)
	bins := []Bin{{ID: active - 1, ReserveX: new(big.Int), ReserveY: e18},
		{ID: active + 1, ReserveX: e18, ReserveY: new(big.Int)}}
	sim := newTestSim(t, withActiveBin(bins, active), active)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xa", Amount: new(big.Int)}, TokenOut: "0xb"})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: "0xa", Amount: new(big.Int)},
		TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})

	_, err = sim.findBinArrIndex(active)
	require.NoError(t, err)
	for _, in := range []string{"0xa", "0xb"} {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(1000)}, TokenOut: otherToken[in]})
		require.NoError(t, err, "%s in", in)
	}
}
