package uniswapv4_test

// Sequential parity (quote, UpdateState, quote) of native CalcAmtOut and legacy CalcAmountOut with real
// swapping hooks. Remove with the legacy methods and the golden data.

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/clanker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/hooktest"
	nftstrat "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/nft_strategy"
	stablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/stable-stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestRealHooksSequentialParity(t *testing.T) {
	t.Parallel()
	const sqrtP = "566274760763156353397804"
	cases := []struct {
		name  string
		addr  common.Address
		extra any
	}{
		{"clanker-static", clanker.StaticFeeHookAddresses[1],
			map[string]any{"p": 1000, "c": 3000, "f": 5000, "0": true, "t": true}},
		{"clanker-static-reversed", clanker.StaticFeeHookAddresses[1],
			map[string]any{"p": 1000, "c": 10000, "f": 500, "t": true}},
		{"clanker-dynamic", clanker.DynamicFeeHookAddresses[1],
			map[string]any{"p": 2000, "0": true, "t": true,
				"f": map[string]any{"r": -236988, "t": -236988, "s": "1753071293", "l": "1753071293", "p": "5"},
				"c": map[string]any{"b": 5000, "m": 50000, "r": "30", "p": "120", "f": 200, "c": "500000000", "d": "7500"}}},
		{"stable-stable-peg", stablestable.HookAddresses[1],
			map[string]any{"k": 1000000, "o": 500, "tm": 50, "rsp": sqrtP, "df": 0, "sap": "0", "bn": 1, "tb": 2}},
		{"stable-stable-off-peg", stablestable.HookAddresses[1],
			map[string]any{"k": 1000000, "o": 500, "tm": 50, "rsp": "570000000000000000000000",
				"df": 1000000000001, "sap": "0", "bn": 1, "tb": 2}},
		{"nft-strategy-fixed-fee", nftstrat.HookAddresses[0], map[string]any{"f": 800}},
		{"nft-strategy-default-fee", nftstrat.HookAddresses[0], map[string]any{}},
	}
	amounts := []string{"1", "1000", "1000000000000", "1000000000000000000", "20000000000000000000000",
		"900000000000000000000000000", "99999999999999999999999999999999999999999999999999"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			base := hooktest.NewPoolSimulator(t, tc.addr, tc.extra)
			tokens := base.GetTokens()
			successes := 0
			for i := range tokens {
				for _, amt := range amounts {
					successes += sequentialParity(t, base, tokens[i], tokens[1-i], amt)
				}
			}
			require.Positive(t, successes, "no quote succeeded; the parity check proves nothing")
		})
	}
}

// sequentialParity quotes twice on a native clone and a legacy clone, applying UpdateState in between, and
// requires identical amounts, remaining, gas and error text. It returns the number of successful quotes.
func sequentialParity(t *testing.T, base *uniswapv4.PoolSimulator, tokenIn, tokenOut, amt string) int {
	t.Helper()
	nativeSim := base.CloneState().(*uniswapv4.PoolSimulator)
	legacySim := base.CloneState().(*uniswapv4.PoolSimulator)
	amtBig, _ := new(big.Int).SetString(amt, 10)
	amtU, overflow := uint256.FromBig(amtBig)
	require.False(t, overflow)
	successes := 0
	for step := 0; step < 3; step++ {
		name := tokenIn + " amt=" + amt + " step=" + string(rune('0'+step))
		lRes, lErr := legacySim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amtBig)}, TokenOut: tokenOut})
		nRes, nErr := nativeSim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU}, TokenOut: tokenOut})
		if lErr != nil || nErr != nil {
			require.Error(t, lErr, name)
			require.Error(t, nErr, name)
			require.Equal(t, lErr.Error(), nErr.Error(), name)
			return successes
		}
		successes++
		require.Equal(t, lRes.TokenAmountOut.Amount.String(), nRes.TokenAmountOut.Amount.Dec(), name+" out")
		require.Equal(t, lRes.RemainingTokenAmountIn.Amount.String(), nRes.RemainingTokenAmountIn.Amount.Dec(), name+" remaining")
		require.Equal(t, lRes.Gas, nRes.Gas, name+" gas")
		require.Equal(t, lRes.TokenAmountOut.Token, nRes.TokenAmountOut.Token, name+" token")
		require.Equal(t, lRes.SwapInfo, nRes.SwapInfo, name+" swapinfo")

		nativeSim.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU},
			TokenAmountOut: *nRes.TokenAmountOut, SwapInfo: nRes.SwapInfo})
		legacySim.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU},
			TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: *uint256.MustFromBig(lRes.TokenAmountOut.Amount)},
			SwapInfo:       lRes.SwapInfo})
	}
	return successes
}
