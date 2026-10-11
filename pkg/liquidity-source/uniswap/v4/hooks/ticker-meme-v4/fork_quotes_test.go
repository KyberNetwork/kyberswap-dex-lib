package tickermemev4

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestDeployedHookForkQuotes(t *testing.T) {
	// Captured from the deployed hook and real Monad PoolManager on a local fork
	// at block 112361144. Every scenario used the same initial price and liquidity.
	trades := []struct {
		zeroForOne        bool
		input             int64
		afterSqrtPriceX96 string
		afterTick         int
	}{
		{true, 1000000, "79228083444637059845758224270", -1},
		{false, 2000000, "79228241584049438317376060983", 0},
		{true, 100, "79228241576205834573415146903", 0},
		{false, 1000000000, "79307311282395070382333503765", 19},
	}
	cases := []struct {
		name                string
		native              bool
		hookFee, creatorTax uint16
		netOutputs          [4]string
	}{
		{"erc20", false, 100, 133, [4]string{"974747", "1949494", "98", "973772831"}},
		{"native", true, 100, 133, [4]string{"974747", "1949494", "98", "973772831"}},
		{"rounding", false, 100, 100, [4]string{"978041", "1956081", "99", "977062940"}},
		{"zero-fees", false, 0, 0, [4]string{"997999", "1995999", "99", "997003000"}},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			p := newPool(t, terms(scenario.hookFee, scenario.creatorTax), scenario.native)
			require.Equal(t, "79228162514264337593543950336", p.V3Pool.SqrtRatioX96.String())
			require.Zero(t, p.V3Pool.TickCurrent)
			for i, trade := range trades {
				index := 1
				if trade.zeroForOne {
					index = 0
				}
				input := big.NewInt(trade.input)
				param := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Info.Tokens[index], Amount: input}, TokenOut: p.Info.Tokens[1-index]}
				result, err := p.CalcAmountOut(param)
				require.NoError(t, err)
				require.Equal(t, scenario.netOutputs[i], result.TokenAmountOut.Amount.String(), "net output differs from deployed hook execution")
				reverse, err := p.CalcAmountIn(pool.CalcAmountInParams{TokenIn: param.TokenAmountIn.Token, TokenAmountOut: *result.TokenAmountOut})
				require.NoError(t, err)
				require.LessOrEqual(t, reverse.TokenAmountIn.Amount.Cmp(input), 0)
				replay, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *reverse.TokenAmountIn, TokenOut: param.TokenOut})
				require.NoError(t, err)
				require.GreaterOrEqual(t, replay.TokenAmountOut.Amount.Cmp(result.TokenAmountOut.Amount), 0)
				p.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: param.TokenAmountIn, TokenAmountOut: *result.TokenAmountOut, SwapInfo: result.SwapInfo})
				require.Equal(t, trade.afterSqrtPriceX96, p.V3Pool.SqrtRatioX96.String())
				require.Equal(t, trade.afterTick, p.V3Pool.TickCurrent)
			}
		})
	}
}
