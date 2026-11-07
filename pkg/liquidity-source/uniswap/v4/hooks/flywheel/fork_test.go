package flywheel

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Recorded on an isolated fork of Robinhood mainnet. Expected outputs were
// checked against both the deployed V4Quoter and actual UniversalRouter receipts.
// The ordinary test suite never opens an RPC connection or signs a transaction.
func TestForkQuoteAndExecutionParity(t *testing.T) {
	data, err := os.ReadFile("testdata/fork-fixtures.json")
	require.NoError(t, err)
	var proof struct {
		ChainID             int `json:"chainId"`
		MainnetTransactions int `json:"mainnetTransactions"`
		Fixtures            []struct {
			Label         string      `json:"label"`
			Pool          entity.Pool `json:"pool"`
			TokenIn       string      `json:"tokenIn"`
			TokenOut      string      `json:"tokenOut"`
			AmountIn      string      `json:"amountIn"`
			AmountOut     string      `json:"amountOut"`
			NextAmountOut string      `json:"nextAmountOut"`
			PostPool      entity.Pool `json:"postPool"`
			AfterSwapGas  int64       `json:"afterSwapGas"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(data, &proof))
	require.Equal(t, 4663, proof.ChainID)
	require.Zero(t, proof.MainnetTransactions)
	require.GreaterOrEqual(t, len(proof.Fixtures), 10)
	for _, f := range proof.Fixtures {
		t.Run(f.Label, func(t *testing.T) {
			sim, err := v4.NewPoolSimulator(f.Pool, valueobject.ChainIDRobinhood)
			require.NoError(t, err)
			args := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: f.TokenIn, Amount: bignumber.NewBig10(f.AmountIn)}, TokenOut: f.TokenOut}
			out, err := sim.CalcAmountOut(args)
			require.NoError(t, err)
			require.Equal(t, f.AmountOut, out.TokenAmountOut.Amount.String())
			require.True(t, out.RemainingTokenAmountIn == nil || out.RemainingTokenAmountIn.Amount.Sign() == 0)
			require.LessOrEqual(t, f.AfterSwapGas, gasAfterSwap)
			for i := 0; i < 3; i++ {
				repeat, err := sim.CalcAmountOut(args)
				require.NoError(t, err)
				require.Equal(t, out, repeat, "quoting must be pure")
			}
			// Reverse exact-input, then execute that input through the simulator.
			in, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: *out.TokenAmountOut, TokenIn: f.TokenIn})
			require.NoError(t, err)
			inverseArgs := args
			inverseArgs.TokenAmountIn.Amount = in.TokenAmountIn.Amount
			inverse, err := sim.CalcAmountOut(inverseArgs)
			require.NoError(t, err)
			require.GreaterOrEqual(t, inverse.TokenAmountOut.Amount.Cmp(out.TokenAmountOut.Amount), 0)
			inverseArgs.TokenAmountIn.Amount = new(big.Int).Sub(in.TokenAmountIn.Amount, big.NewInt(1))
			less, err := sim.CalcAmountOut(inverseArgs)
			if err == nil {
				require.Less(t, less.TokenAmountOut.Amount.Cmp(out.TokenAmountOut.Amount), 0)
			}
			clone := sim.CloneState().(*v4.PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: args.TokenAmountIn, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee, SwapInfo: out.SwapInfo})
			var post v3.ExtraTickU256
			require.NoError(t, json.Unmarshal([]byte(f.PostPool.Extra), &post))
			require.Equal(t, *post.SqrtPriceX96, clone.V3Pool.SqrtRatioX96)
			require.Equal(t, *post.Liquidity, clone.V3Pool.Liquidity)
			require.Equal(t, *post.Tick, clone.V3Pool.TickCurrent)
			next, err := clone.CalcAmountOut(args)
			require.NoError(t, err)
			require.Equal(t, f.NextAmountOut, next.TokenAmountOut.Amount.String())
			untouched, err := sim.CalcAmountOut(args)
			require.NoError(t, err)
			require.Equal(t, out, untouched)
		})
	}
}
