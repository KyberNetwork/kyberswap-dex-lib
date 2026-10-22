package peach

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type liveCase struct {
	Name    string
	ChainID valueobject.ChainID
	Pool    entity.Pool
	Quotes  []struct {
		ZeroForOne            bool
		AmountIn, AmountOut   string
		Gas                   int64
		InsufficientLiquidity bool
	}
}

func loadLiveCases(t testing.TB) []liveCase {
	t.Helper()
	data, err := os.ReadFile("testdata/live_pools.json")
	require.NoError(t, err)
	var fixture struct{ Cases []liveCase }
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Len(t, fixture.Cases, 5)
	for i := range fixture.Cases {
		var extra uniswapv4.Extra
		require.NoError(t, json.Unmarshal([]byte(fixture.Cases[i].Pool.Extra), &extra))
		r0, r1 := uniswapv4.EstimateReservesFromTicks(extra.SqrtPriceX96, extra.Ticks)
		fixture.Cases[i].Pool.Reserves = []string{r0.String(), r1.String()}
	}
	return fixture.Cases
}

func TestLiveFixedBlockQuoter(t *testing.T) {
	for _, c := range loadLiveCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			sim, err := uniswapv4.NewPoolSimulator(c.Pool, c.ChainID)
			require.NoError(t, err)
			for i, q := range c.Quotes {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					input, output := 0, 1
					if !q.ZeroForOne {
						input, output = 1, 0
					}
					amount, ok := new(big.Int).SetString(q.AmountIn, 10)
					require.True(t, ok)
					params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: c.Pool.Tokens[input].Address, Amount: amount}, TokenOut: c.Pool.Tokens[output].Address}
					result, err := sim.CalcAmountOut(params)
					if q.InsufficientLiquidity {
						// The canonical Quoter insists on filling the entire input. The parent v4
						// simulator may instead expose its partial fill through RemainingTokenAmountIn.
						if err != nil {
							require.True(t, errors.Is(err, uniswapv3.ErrInsufficientBalance) || errors.Is(err, uniswapv3.ErrAtOrAboveLargest) || errors.Is(err, uniswapv3.ErrBelowSmallest) || errors.Is(err, uniswapv3.ErrZeroAmount), "unexpected liquidity error: %v", err)
						} else {
							require.Positive(t, result.TokenAmountOut.Amount.Sign())
							require.NotNil(t, result.RemainingTokenAmountIn)
							require.Negative(t, result.RemainingTokenAmountIn.Amount.Cmp(amount))
							require.Positive(t, result.RemainingTokenAmountIn.Amount.Sign())
						}
						return
					}
					want, ok := new(big.Int).SetString(q.AmountOut, 10)
					require.True(t, ok)
					if want.Sign() == 0 {
						require.ErrorIs(t, err, uniswapv3.ErrZeroAmount)
						return
					}
					require.NoError(t, err)
					require.Equal(t, want, result.TokenAmountOut.Amount)
					if result.RemainingTokenAmountIn != nil {
						require.Zero(t, result.RemainingTokenAmountIn.Amount.Sign())
					}
					require.Positive(t, result.Gas)
					t.Logf("chain=%d block=%d in=%s out=%s simulatorGas=%d quoterGas=%d", c.ChainID, c.Pool.BlockNumber, q.AmountIn, q.AmountOut, result.Gas, q.Gas)
					// Every quote is speculative: repeat against the same pool and compare.
					again, err := sim.CalcAmountOut(params)
					require.NoError(t, err)
					require.Equal(t, result.TokenAmountOut, again.TokenAmountOut)
				})
			}
		})
	}
}

func TestLivePoolSequenceUsesSimulatedPrice(t *testing.T) {
	for _, c := range loadLiveCases(t) {
		if c.ChainID != 5042 || c.Name != "arc-Ramp-pool" {
			continue
		}
		sim, err := uniswapv4.NewPoolSimulator(c.Pool, c.ChainID)
		require.NoError(t, err)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: c.Pool.Tokens[0].Address, Amount: big.NewInt(100)}, TokenOut: c.Pool.Tokens[1].Address}
		first, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.NotNil(t, first.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo)
		clone := sim.CloneState().(*uniswapv4.PoolSimulator)
		clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})
		second, err := clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: params.TokenOut, Amount: first.TokenAmountOut.Amount}, TokenOut: params.TokenAmountIn.Token})
		require.NoError(t, err)
		require.Nil(t, second.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo)
		require.Greater(t, second.Gas, first.Gas)
		again, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, first.TokenAmountOut, again.TokenAmountOut)
	}
}
