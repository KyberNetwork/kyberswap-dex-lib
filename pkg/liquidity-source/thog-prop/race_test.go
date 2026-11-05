package thogprop

import (
	"math/big"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type routeStep struct {
	out *big.Int
	err error
}

// replayRoute quotes every step on sim and applies each successful SwapInfo, the way
// the router walks a multi-hop path on one cloned pool.
func replayRoute(sim *PoolSimulator, steps []pool.CalcAmountOutParams) []routeStep {
	got := make([]routeStep, len(steps))
	for i, params := range steps {
		r, err := sim.quoteAtCapture(params)
		if got[i].err = err; err == nil {
			got[i].out = r.TokenAmountOut.Amount
			sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: r.SwapInfo})
		}
	}
	return got
}

// TestCloneStateConcurrentRoutesAreIsolated runs the same multi-hop sequence on many
// clones at once while the base keeps quoting. Under -race this proves CloneState
// shares nothing UpdateBalance writes; the outputs prove clones never see each
// other's updates and the base never sees any.
func TestCloneStateConcurrentRoutesAreIsolated(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	s := stateFrom(t, f.Snapshots[0])
	base, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)

	var steps []pool.CalcAmountOutParams
	for _, c := range f.Snapshots[0].Cases {
		if c.Method == "execution" && c.Success {
			amount := number(t, c.Amount)
			steps = append(steps, pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount.ToBig()},
				TokenOut:      s.Tokens[c.OutputIndex].Address,
			})
		}
	}
	require.GreaterOrEqual(t, len(steps), 8)
	// Repeat the sequence so later steps run against depleted inventory.
	steps = append(steps, steps...)

	want := replayRoute(base.CloneState().(*PoolSimulator), steps)
	applied := 0
	for _, w := range want {
		if w.err == nil {
			applied++
		}
	}
	require.Positive(t, applied, "the route must apply updates")
	fresh := replayRoute(base.CloneState().(*PoolSimulator), steps[:1])
	require.Equal(t, want[0], fresh[0])

	stateBefore, reservesBefore := base.state, slices.Clone(base.Info.Reserves)
	var wg sync.WaitGroup
	results := make([][]routeStep, 16)
	for i := range results {
		wg.Go(func() { results[i] = replayRoute(base.CloneState().(*PoolSimulator), steps) })
	}
	for range 4 { // readers of the base pool run beside the clone writers
		wg.Go(func() {
			for _, params := range steps {
				_, _ = base.quoteAtCapture(params)
			}
		})
	}
	wg.Wait()

	for i, got := range results {
		require.Equal(t, want, got, "clone %d diverged", i)
	}
	require.Equal(t, stateBefore, base.state)
	require.Equal(t, reservesBefore, base.Info.Reserves)
}
