package metricpropamm

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testWETH = "0x4200000000000000000000000000000000000006"
	testUSDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

func newTestSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	file, err := os.ReadFile("./pool_state_test.json")
	require.NoError(t, err)
	var state PairData
	require.NoError(t, json.Unmarshal(file, &state))
	extra, reserves, err := convertPoolState(state, &Config{MaxAge: 60, IsV2: true})
	require.NoError(t, err)
	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  "0xa07938ea73e9d8eb23535816d073c2731ea24946",
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{testWETH, testUSDC},
			Reserves: []*big.Int{bignumber.NewBig(reserves[0]), bignumber.NewBig(reserves[1])},
		},
		poolTimestamp: time.Now().Unix(),
		extra:         extra,
		decimalsDiff:  12,
	}
}

func quoteResult(t *testing.T, s pool.IPoolSimulator, tokenIn, tokenOut, amount string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig(amount)},
		TokenOut:      tokenOut,
	})
}

// swap quotes amount on s and applies it, as a route's second use of the pool does.
func swap(t *testing.T, s pool.IPoolSimulator, tokenIn, tokenOut, amount string) *big.Int {
	t.Helper()
	res, err := quoteResult(t, s, tokenIn, tokenOut, amount)
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig(amount)},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
	})
	return res.TokenAmountOut.Amount
}

// After a swap, the pool still quotes the depth it has left: a swap of a then one of b return what one swap
// of a+b returns (up to rounding), and the original is untouched by its clone's swaps.
func TestUpdateBalanceKeepsTheUnfilledBins(t *testing.T) {
	for _, tc := range []struct {
		name, in, out, a, b, ab string
	}{
		{"weth to usdc", testWETH, testUSDC, "10000000000000000", "90000000000000000", "100000000000000000"},
		{"usdc to weth", testUSDC, testWETH, "20000000", "180000000", "200000000"},
		{"a dust swap first", testWETH, testUSDC, "1", "100000000000000000", "100000000000000001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSimulator(t)
			whole, err := quoteResult(t, s, tc.in, tc.out, tc.ab)
			require.NoError(t, err)

			c := s.CloneState()
			first := swap(t, c, tc.in, tc.out, tc.a)
			second, err := quoteResult(t, c, tc.in, tc.out, tc.b)
			require.NoError(t, err, "the pool quotes nothing after one swap")

			got := new(big.Int).Add(first, second.TokenAmountOut.Amount)
			diff := new(big.Int).Sub(got, whole.TokenAmountOut.Amount)
			diff.Abs(diff)
			// rounding: a few units of the output per swap, far below 1e-6 of it
			limit := new(big.Int).Div(whole.TokenAmountOut.Amount, big.NewInt(1_000_000))
			require.True(t, diff.Cmp(limit) <= 0, "a then b returns %s, a+b returns %s", got, whole.TokenAmountOut.Amount)

			again, err := quoteResult(t, s, tc.in, tc.out, tc.ab)
			require.NoError(t, err)
			require.Equal(t, whole.TokenAmountOut.Amount.String(), again.TokenAmountOut.Amount.String(),
				"the clone's swap changed the original")
		})
	}
}

// A swap that takes every bin leaves the direction with nothing to quote.
func TestUpdateBalanceThroughEveryBin(t *testing.T) {
	s := newTestSimulator(t)
	last := s.extra.Bids[len(s.extra.Bids)-1].CumulativeVolume
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: big.NewInt(1)},
		TokenAmountOut: pool.TokenAmount{Token: testUSDC, Amount: new(big.Int).Set(last)},
	})
	require.Empty(t, s.extra.Bids)
	_, err := quoteResult(t, s, testWETH, testUSDC, "1000000000000000")
	require.Error(t, err)
}
