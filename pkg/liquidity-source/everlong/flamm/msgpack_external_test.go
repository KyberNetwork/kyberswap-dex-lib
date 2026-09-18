package everlongflamm_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	everlongflamm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/everlong/flamm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The pool-service -> router-service hop through the PRODUCTION encoder (pkg/msgpack EncodePoolSimulatorsMap /
// DecodePoolSimulatorsMap over the pool.IPoolSimulator interface): a simulator that has already adopted a mixed
// sequence of fills, and one that is broken, decode to simulators that quote, adopt and refuse identically.

func swapParams(sim *everlongflamm.PoolSimulator, sell bool, a uint64) pool.CalcAmountOutParams {
	in, out := sim.Info.Tokens[0], sim.Info.Tokens[1]
	if !sell {
		in, out = out, in
	}
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: new(big.Int).SetUint64(a)},
		TokenOut: out}
}

func renderResult(t *testing.T, sim *everlongflamm.PoolSimulator, params pool.CalcAmountOutParams) string {
	r, err := sim.CalcAmountOut(params)
	if err != nil {
		return "err " + err.Error()
	}
	si := r.SwapInfo.(everlongflamm.SwapInfo)
	return fmt.Sprintf("%s/%s/%s/%d/%d/%v/%s/%s/%s/%s", r.TokenAmountOut.Amount, r.RemainingTokenAmountIn.Amount,
		r.Fee.Amount, r.Gas, si.Venue, si.PoolAssetIn, si.AmountInUsed.Dec(), si.AmountOut.Dec(), si.SpreadPpm.Dec(),
		everlongflamm.ExportNextJSON(t, r))
}

func productionRoundTrip(t *testing.T, sim *everlongflamm.PoolSimulator) *everlongflamm.PoolSimulator {
	raw, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{sim.Info.Address: sim})
	require.NoError(t, err)
	m, err := msgpack.DecodePoolSimulatorsMap(raw)
	require.NoError(t, err)
	dec, ok := m[sim.Info.Address].(*everlongflamm.PoolSimulator)
	require.True(t, ok, "decoded %T", m[sim.Info.Address])
	return dec
}

func TestMsgpackProductionEncoder(t *testing.T) {
	policy := everlongflamm.Policy{LeverRouting: true}
	sim, err := everlongflamm.NewPoolSimulator(everlongflamm.ExportEntity(t, "51313000", "armed", policy))
	require.NoError(t, err)
	ts := everlongflamm.ExportTimestamp(sim)
	everlongflamm.ExportSetClock(sim, ts)

	// Adopt a mixed sequence first (both venues, both directions, a later clock).
	steps := []struct {
		sell  bool
		a     uint64
		venue int
	}{{true, 5_000, 1}, {false, 20_000_000, 1}, {true, 60_000, -1}, {false, 3_000_000, 0}, {true, 2_000, 1},
		{false, 150_000_000, -1}}
	venues := map[string]int{}
	for i, s := range steps {
		everlongflamm.ExportSetClock(sim, ts+uint64(12*(i+1)))
		r, err := everlongflamm.ExportCalcVenue(sim, swapParams(sim, s.sell, s.a), s.venue)
		require.NoError(t, err, "step %d", i)
		venues[fmt.Sprintf("venue%d/sell=%v", r.SwapInfo.(everlongflamm.SwapInfo).Venue, s.sell)]++
		sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	}
	seq, broken := everlongflamm.ExportSeq(sim)
	require.Equal(t, uint64(len(steps)), seq)
	require.Equal(t, 2, venues["venue1/sell=true"])
	require.Equal(t, 1, venues["venue1/sell=false"])
	require.False(t, broken)
	t.Logf("adopted %d fills: %v", seq, venues)

	dec := productionRoundTrip(t, sim)
	require.Equal(t, everlongflamm.ExportStateJSON(t, sim), everlongflamm.ExportStateJSON(t, dec))
	dseq, dbroken := everlongflamm.ExportSeq(dec)
	require.Equal(t, seq, dseq)
	require.False(t, dbroken)
	require.Equal(t, sim.GetMetaInfo("", ""), dec.GetMetaInfo("", ""))
	require.Equal(t, sim.Info.Reserves, dec.Info.Reserves)
	require.Equal(t, sim.Policy, dec.Policy)

	clock := everlongflamm.ExportTimestamp(sim) + 5
	everlongflamm.ExportSetClock(sim, clock)
	everlongflamm.ExportSetClock(dec, clock)
	for _, sell := range []bool{true, false} {
		lo, hi := uint64(1), uint64(3_000_000)
		if !sell {
			lo, hi = 100, 20_000_000_000
		}
		for i := 0; i < 40; i++ {
			a := lo + (hi-lo)*uint64(i*i)/1600
			want := renderResult(t, sim, swapParams(sim, sell, a))
			got := renderResult(t, dec, swapParams(dec, sell, a))
			require.Equal(t, want, got, "sell=%v a=%d", sell, a)
		}
	}

	// Both continue identically, and a SwapInfo quoted before encoding adopts on the decoded copy (same seq).
	r, err := sim.CalcAmountOut(swapParams(sim, true, 7_000))
	require.NoError(t, err)
	dec.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
	require.Equal(t, everlongflamm.ExportStateJSON(t, sim), everlongflamm.ExportStateJSON(t, dec))

	// A broken simulator stays broken through the hop.
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo}) // stale seq
	_, err = sim.CalcAmountOut(swapParams(sim, true, 7_000))
	require.ErrorIs(t, err, everlongflamm.ErrSwapInfoMismatch)
	dec2 := productionRoundTrip(t, sim)
	everlongflamm.ExportSetClock(dec2, clock)
	_, err = dec2.CalcAmountOut(swapParams(dec2, true, 7_000))
	require.ErrorIs(t, err, everlongflamm.ErrSwapInfoMismatch)
}
