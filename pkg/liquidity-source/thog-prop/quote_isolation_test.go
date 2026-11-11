package thogprop

import (
	"context"
	"encoding/json"
	"math/big"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func isolationSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	// The test clock is frozen; exercise the public live quote method.
	sim.capturedAtMs = time.Now().UnixMilli()
	return sim
}

func simulatorImage(t *testing.T, sim *PoolSimulator) []byte {
	t.Helper()
	data, err := json.Marshal(&struct {
		State State
		Info  pool.PoolInfo
	}{sim.state, sim.Info})
	require.NoError(t, err)
	return data
}

func TestPairQuoteErrorsLeaveEveryOtherPairUnchanged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sim := isolationSimulator(t)
		before := simulatorImage(t, sim)
		type observation struct {
			params pool.CalcAmountOutParams
			result *pool.CalcAmountOutResult
			err    error
		}
		var baseline []observation
		successes := 0
		for i, input := range sim.state.Tokens {
			for o, output := range sim.state.Tokens {
				if i == o {
					continue
				}
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: input.Address, Amount: powers[input.Decimals].ToBig()}, TokenOut: output.Address}
				result, err := sim.CalcAmountOut(params)
				baseline = append(baseline, observation{params, result, err})
				if err == nil {
					successes++
				}
			}
		}
		require.Greater(t, successes, 2)
		input, output := sim.state.Tokens[0].Address, sim.state.Tokens[3].Address
		for _, params := range []pool.CalcAmountOutParams{
			{TokenAmountIn: pool.TokenAmount{Token: input}, TokenOut: output},
			{TokenAmountIn: pool.TokenAmount{Token: input, Amount: big.NewInt(-1)}, TokenOut: output},
			{TokenAmountIn: pool.TokenAmount{Token: input, Amount: new(big.Int).Lsh(big.NewInt(1), 256)}, TokenOut: output},
			{TokenAmountIn: pool.TokenAmount{Token: input, Amount: maxUint.ToBig()}, TokenOut: output},
			{TokenAmountIn: pool.TokenAmount{Token: input, Amount: big.NewInt(0)}, TokenOut: output},
			{TokenAmountIn: pool.TokenAmount{Token: input, Amount: big.NewInt(1_000_000)}, TokenOut: input},
			{TokenAmountIn: pool.TokenAmount{Token: "0x0000000000000000000000000000000000000001", Amount: big.NewInt(1)}, TokenOut: output},
		} {
			result, err := sim.CalcAmountOut(params)
			require.Error(t, err)
			require.Nil(t, result)
			for _, want := range baseline {
				got, err := sim.CalcAmountOut(want.params)
				require.Equal(t, want.err, err)
				require.Equal(t, want.result, got)
			}
			require.Equal(t, before, simulatorImage(t, sim))
		}
		t.Logf("All %d directed-pair outcomes preserved after each of seven failed quote requests", len(baseline))
	})
}

func TestConcurrentPairFailuresDoNotAffectSuccessfulQuotes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sim := isolationSimulator(t)
		before := simulatorImage(t, sim)
		good := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: sim.state.Tokens[0].Address, Amount: big.NewInt(1_000_000)}, TokenOut: sim.state.Tokens[3].Address}
		want, err := sim.CalcAmountOut(good)
		require.NoError(t, err)
		bad := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: sim.state.Tokens[0].Address, Amount: maxUint.ToBig()}, TokenOut: sim.state.Tokens[2].Address}
		var workers sync.WaitGroup
		for worker := 0; worker < 8; worker++ {
			workers.Go(func() {
				for attempt := 0; attempt < 32; attempt++ {
					failed, err := sim.CalcAmountOut(bad)
					if err == nil || failed != nil {
						t.Errorf("oversized quote unexpectedly succeeded")
						return
					}
					got, err := sim.CalcAmountOut(good)
					if err != nil || got == nil || got.TokenAmountOut.Amount.Cmp(want.TokenAmountOut.Amount) != 0 {
						t.Errorf("other pair changed after concurrent failure: %v", err)
						return
					}
				}
			})
		}
		workers.Wait()
		require.Equal(t, before, simulatorImage(t, sim))
	})
}

// Inject a pair-local bug at the public interface to exercise Kyber's existing
// per-call panic boundary, without adding another guard to the adapter hot path.
type pairPanicSimulator struct {
	*PoolSimulator
	input, output string
}

func (s *pairPanicSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == s.input && params.TokenOut == s.output {
		panic("injected pair quote bug")
	}
	return s.PoolSimulator.CalcAmountOut(params)
}

func TestKyberQuoteWrapperRecoversOnePairAndContinuesOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sim := isolationSimulator(t)
		before := simulatorImage(t, sim)
		wrapped := &pairPanicSimulator{PoolSimulator: sim, input: sim.state.Tokens[0].Address, output: sim.state.Tokens[2].Address}
		amount := pool.TokenAmount{Token: wrapped.input, Amount: big.NewInt(1_000_000)}
		goodOutput := sim.state.Tokens[3].Address
		want, err := pool.CalcAmountOut(context.Background(), wrapped, amount, goodOutput, nil)
		require.NoError(t, err)
		failed, err := pool.CalcAmountOut(context.Background(), wrapped, amount, wrapped.output, nil)
		require.ErrorContains(t, err, "injected pair quote bug")
		require.Nil(t, failed)
		got, err := pool.CalcAmountOut(context.Background(), wrapped, amount, goodOutput, nil)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, before, simulatorImage(t, sim))
	})
}
