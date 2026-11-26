package testutil

import (
	"math/big"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// CloneOp changes the state of sim, a clone, as a swap or a liquidity operation of a router does.
type CloneOp func(t testing.TB, sim pool.IPoolSimulator)

// SwapOp quotes each of params on sim in turn and applies each swap with UpdateBalance.
func SwapOp(params ...pool.CalcAmountOutParams) CloneOp {
	return func(t testing.TB, sim pool.IPoolSimulator) {
		for _, p := range params {
			res, err := sim.CalcAmountOut(p)
			if err != nil {
				t.Error(err)
				return
			}
			sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: p.TokenAmountIn,
				TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo, SwapLimit: p.Limit})
		}
	}
}

// TestCloneIsolation checks that each op applied to a clone of orig changes only the clone:
//   - one op moves the clone's reserves and leaves the reserves of orig and of each watched simulator (a base
//     pool orig shares with others, say) as they were;
//   - ops on clones on 2 goroutines, while a third reads orig's reserves, share no writes: the reads see orig's
//     reserves throughout, and -race reports nothing.
func TestCloneIsolation(t *testing.T, orig pool.IPoolSimulator, ops map[string]CloneOp,
	watched ...pool.IPoolSimulator) {
	t.Helper()
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			before := reserveStrings(orig.GetReserves())
			watchedBefore := make([][]string, len(watched))
			for i, w := range watched {
				watchedBefore[i] = reserveStrings(w.GetReserves())
			}

			clone := orig.CloneState()
			require.NotNil(t, clone, "CloneState returned nil")
			op(t, clone)
			require.NotEqual(t, before, reserveStrings(clone.GetReserves()), "the op did not move the clone's reserves")
			require.Equal(t, before, reserveStrings(orig.GetReserves()), "an op on a clone moved the original's reserves")
			for i, w := range watched {
				require.Equal(t, watchedBefore[i], reserveStrings(w.GetReserves()),
					"an op on a clone moved watched simulator %d's reserves", i)
			}

			var wg sync.WaitGroup
			for range 2 {
				wg.Go(func() {
					for range 50 {
						op(t, orig.CloneState())
					}
				})
			}
			wg.Go(func() {
				for range 200 {
					if got := reserveStrings(orig.GetReserves()); !slices.Equal(got, before) {
						t.Errorf("read the original's reserves %v while clones changed, want %v", got, before)
						return
					}
				}
			})
			wg.Wait()
		})
	}
}

func reserveStrings(rs []*big.Int) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}
