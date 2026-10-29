package integral

import (
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The tests below quote the fixtures of TestCalcAmountMatchesChain on one simulator many times, as a
// router does. A quote must not change the fee factors that later quotes take.

// TestQuotesOnOneSimulatorMatchChain quotes every fixture amount on one simulator, twice over.
func TestQuotesOnOneSimulatorMatchChain(t *testing.T) {
	for name, fixture := range loadOnchainFeeFixtures(t) {
		t.Run(name, func(t *testing.T) {
			setBlockTimestamp(t, fixture.BlockTimestamp)
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)
			for range 2 {
				for _, q := range fixture.Quotes {
					got, err := fixture.quote(sim, q)
					if assert.NoError(t, err, q) {
						assert.Equal(t, q.Result, got, q)
					}
				}
			}
		})
	}
}

// TestQuotesInTheSecondOfTheLastTimepoint quotes in the second the plugin wrote its last timepoint
// in, where a swap writes none. Every quote on one simulator must be what a new simulator quotes.
func TestQuotesInTheSecondOfTheLastTimepoint(t *testing.T) {
	for name, fixture := range loadOnchainFeeFixtures(t) {
		t.Run(name, func(t *testing.T) {
			first, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)
			setBlockTimestamp(t, first.volatilityOracle.LastTimepointTimestamp)
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)
			for _, q := range fixture.Quotes {
				fresh, err := NewPoolSimulator(fixture.Pool)
				require.NoError(t, err)
				want, err := fixture.quote(fresh, q)
				require.NoError(t, err, q)
				for range 20 {
					got, err := fixture.quote(sim, q)
					require.NoError(t, err, q)
					require.Equal(t, want, got, q)
				}
			}
		})
	}
}

// TestCloneLeavesTheOriginal swaps on a clone of a simulator, the largest fixture amount each way,
// and quotes the clone after each swap. The original must still quote as the chain did.
func TestCloneLeavesTheOriginal(t *testing.T) {
	for name, fixture := range loadOnchainFeeFixtures(t) {
		t.Run(name, func(t *testing.T) {
			setBlockTimestamp(t, fixture.BlockTimestamp)
			sim, err := NewPoolSimulator(fixture.Pool)
			require.NoError(t, err)

			clone := sim.CloneState().(*PoolSimulator)
			token0, token1 := fixture.Pool.Tokens[0].Address, fixture.Pool.Tokens[1].Address
			for _, zeroToOne := range []bool{true, false, true} {
				in, out := token0, token1
				if !zeroToOne {
					in, out = token1, token0
				}
				var largest *big.Int
				for _, q := range fixture.Quotes {
					amount, _ := new(big.Int).SetString(q.Amount, 10)
					if q.Kind == "in" && q.ZeroToOne == zeroToOne && (largest == nil || amount.Cmp(largest) > 0) {
						largest = amount
					}
				}
				require.NotNil(t, largest)
				res, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: in, Amount: largest}, TokenOut: out})
				require.NoError(t, err)
				clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
				for _, q := range fixture.Quotes {
					_, _ = fixture.quote(clone, q)
				}
			}

			for _, q := range fixture.Quotes {
				got, err := fixture.quote(sim, q)
				if assert.NoError(t, err, q) {
					assert.Equal(t, q.Result, got, q)
				}
			}
		})
	}
}

// TestFirstQuotesConcurrently quotes a new simulator from many goroutines at once.
func TestFirstQuotesConcurrently(t *testing.T) {
	for name, fixture := range loadOnchainFeeFixtures(t) {
		t.Run(name, func(t *testing.T) {
			setBlockTimestamp(t, fixture.BlockTimestamp)
			for range 10 {
				sim, err := NewPoolSimulator(fixture.Pool)
				require.NoError(t, err)
				var wg sync.WaitGroup
				for _, q := range fixture.Quotes[:min(8, len(fixture.Quotes))] {
					wg.Go(func() {
						got, err := fixture.quote(sim, q)
						if assert.NoError(t, err, q) {
							assert.Equal(t, q.Result, got, q)
						}
					})
				}
				wg.Wait()
			}
		})
	}
}
