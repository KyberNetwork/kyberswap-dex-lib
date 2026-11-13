package carbon

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtOutParity checks CalcAmtOut against CalcAmountOut on every output field and error.
// The uint256 path must not drift from big.Int math at any amount, including the largest order size.
func TestCalcAmtOutParity(t *testing.T) {
	t.Parallel()

	const (
		weth = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		usdc = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		gvnr = "0xfc60fc0145d7330e5abcfc52af7b043a1ce18e7d"
	)

	cases := []struct {
		name     string
		poolJSON string
		tokenA   string
		tokenB   string
	}{
		{"WETHUSDC", wethUSDCPool, weth, usdc},
		{"GVNRETH", gvnrETHPool, "", gvnr},
	}

	for _, tc := range cases {
		sim := mustNewSim(t, tc.poolJSON)

		var maxOrder uint256.Int
		for _, strategy := range sim.Strategies {
			for _, order := range strategy.Orders {
				if order.Y != nil && order.Y.Gt(&maxOrder) {
					maxOrder.Set(order.Y)
				}
				if order.Z != nil && order.Z.Gt(&maxOrder) {
					maxOrder.Set(order.Z)
				}
			}
		}
		maxMinusOne := new(uint256.Int).Sub(&maxOrder, uint256.NewInt(1))

		amounts := []*uint256.Int{
			uint256.NewInt(0),
			uint256.NewInt(1),
			uint256.NewInt(1000),
			uint256.NewInt(1e9),
			uint256.MustFromDecimal("1000000000000000000"),
			uint256.MustFromDecimal("1000000000000000000000000"),
			maxMinusOne,
			&maxOrder,
		}

		directions := [][2]string{{tc.tokenA, tc.tokenB}, {tc.tokenB, tc.tokenA}}
		for _, dir := range directions {
			for _, amount := range amounts {
				name := fmt.Sprintf("%s/%s->%s/%s", tc.name, dir[0], dir[1], amount.Dec())
				t.Run(name, func(t *testing.T) {
					checkCalcAmtOutParity(t, sim, dir[0], dir[1], amount)
				})
			}
		}
	}
}

func checkCalcAmtOutParity(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amount *uint256.Int) {
	t.Helper()

	want, wantErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount.ToBig()},
		TokenOut:      tokenOut,
	})
	got, gotErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amount},
		TokenOut:      tokenOut,
	})

	if wantErr != nil || gotErr != nil {
		require.Equal(t, wantErr, gotErr)
		require.Nil(t, got)
		require.Nil(t, want)
		return
	}

	require.Equal(t, want.TokenAmountOut.Token, got.TokenAmountOut.Token)
	require.Zero(t, want.TokenAmountOut.Amount.Cmp(got.TokenAmountOut.Amount.ToBig()),
		"amount: big=%s uint256=%s", want.TokenAmountOut.Amount, got.TokenAmountOut.Amount.Dec())

	require.Equal(t, want.Fee.Token, got.Fee.Token)
	require.Zero(t, want.Fee.Amount.Cmp(got.Fee.Amount.ToBig()),
		"fee: big=%s uint256=%s", want.Fee.Amount, got.Fee.Amount.Dec())

	require.Equal(t, want.Gas, got.Gas)
	wantSwap, ok := want.SwapInfo.(SwapInfo)
	require.True(t, ok)
	gotSwap, ok := got.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.True(t, reflect.DeepEqual(sortedTradeActions(wantSwap), sortedTradeActions(gotSwap)), "SwapInfo differs")
}

// sortedTradeActions returns a copy ordered by strategy. Match walks ordersMap, so the action order
// changes between calls even for CalcAmountOut; the fills themselves do not.
func sortedTradeActions(s SwapInfo) []TradeAction {
	actions := slices.Clone(s.TradeActions)
	slices.SortFunc(actions, func(a, b TradeAction) int {
		return cmp.Compare(a.StrategyId, b.StrategyId)
	})
	return actions
}
