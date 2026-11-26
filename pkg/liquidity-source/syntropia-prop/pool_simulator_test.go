package syntropiaprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// Pool.quote at the fixture's block, for sizes at and between the lens's levels. The
// simulator must never quote above the pool, and stays within 5 bps of it.
func TestPoolSimulator_CalcAmountOut_AgainstPoolQuote(t *testing.T) {
	t.Parallel()
	simulator := fixtureSimulator(t, 10_000)
	for _, tc := range []struct{ tokenIn, tokenOut, amountIn, poolQuote string }{
		{testWETH, testUSDC, "1000000000000000", "1682602"},
		{testWETH, testUSDC, "500000000000000000", "841249514"},
		{testWETH, testUSDC, "1114009185377069829", "1874174999"},
		{testWETH, testUSDC, "3000000000000000000", "5045916651"},
		{testWETH, testUSDC, "10000000000000000000", "16804994962"},
		{testWETH, testUSDC, "30000000000000000000", "50309807346"},
		{testUSDC, testWETH, "1000000", "593959990731442"},
		{testUSDC, testWETH, "500000000", "296968706739309967"},
		{testUSDC, testWETH, "3000000000", "1781482493716988193"},
		{testUSDC, testWETH, "12000000000", "7121167362798628728"},
		{testUSDC, testWETH, "30000000000", "17779087522502985544"},
	} {
		result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: bi(tc.amountIn)},
			TokenOut:      tc.tokenOut,
		})
		require.NoError(t, err, tc.amountIn)
		got, want := result.TokenAmountOut.Amount, bi(tc.poolQuote)
		require.LessOrEqual(t, got.Cmp(want), 0, "%s in: quoted %s above the pool's %s", tc.amountIn, got, want)
		floor := new(big.Int).Div(new(big.Int).Mul(want, big.NewInt(9_995)), big.NewInt(10_000))
		require.GreaterOrEqual(t, got.Cmp(floor), 0, "%s in: quoted %s, over 5 bps under the pool's %s", tc.amountIn, got, want)
		require.Equal(t, int64(defaultGas), result.Gas)
	}
}

// The pool reverts above its last level (36 WETH and 34,000 USDC both did at the fixture's
// block), so the simulator must refuse those sizes too.
func TestPoolSimulator_CalcAmountOut_AboveDepth(t *testing.T) {
	t.Parallel()
	simulator := fixtureSimulator(t, 10_000)
	for _, tc := range []struct{ tokenIn, tokenOut, amountIn string }{
		{testWETH, testUSDC, "36000000000000000000"},
		{testUSDC, testWETH, "34000000000"},
	} {
		_, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: bi(tc.amountIn)},
			TokenOut:      tc.tokenOut,
		})
		require.ErrorIs(t, err, orderbook.ErrInsufficientLiquidity)
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, fixtureSimulator(t, 9_990))
}

// A second swap must be priced after the first: two halves add up to one whole.
func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	half, whole := bi("5000000000000000000"), bi("10000000000000000000")
	quote := func(s *PoolSimulator, amountIn *big.Int) *pool.CalcAmountOutResult {
		result, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: amountIn},
			TokenOut:      testUSDC,
		})
		require.NoError(t, err)
		return result
	}

	original := fixtureSimulator(t, 10_000)
	split := original.CloneState().(*PoolSimulator)
	first := quote(split, half)
	split.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWETH, Amount: half},
		TokenAmountOut: *first.TokenAmountOut,
	})
	second := quote(split, half)
	require.Less(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), 0)

	sum := new(big.Int).Add(first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
	diff := new(big.Int).Sub(sum, quote(original, whole).TokenAmountOut.Amount)
	require.LessOrEqual(t, diff.CmpAbs(big.NewInt(2)), 0)
	require.Equal(t, first.TokenAmountOut.Amount, quote(original, half).TokenAmountOut.Amount,
		"a clone's swap must not move the original")
}

// Route finding refuses a snapshot older than maxAge; the liquidity indexer never does.
func TestPoolSimulator_StaleCheck(t *testing.T) {
	t.Parallel()
	old := fixturePool(t, 10_000, maxAge+5*time.Second)
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: old, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.ErrorIs(t, err, orderbook.ErrLevelsTooOld)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: old})
	require.NoError(t, err)

	fresh := fixturePool(t, 10_000, 0)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: fresh, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
}

func TestPoolSimulator_MetaInfoAndLimit(t *testing.T) {
	t.Parallel()
	simulator := fixtureSimulator(t, 10_000)
	require.Equal(t, pool.MetaInfo{ApprovalAddress: testPool, BlockNumber: 62603623},
		simulator.GetMetaInfo(testWETH, testUSDC))
	require.Equal(t, map[string]*big.Int{testWETH: fixtureBalanceWETH, testUSDC: fixtureBalanceUSDC},
		simulator.CalculateLimit())
}
