package bin

// Sequential parity (quote, UpdateState, quote) of native CalcAmtOut and legacy CalcAmountOut over the sample
// pools, hook swap fees and Arc native scaling. Remove with the legacy methods and the golden data.

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestCalcAmtOutLegacyParity(t *testing.T) {
	t.Parallel()
	variants := []struct {
		name    string
		file    string
		hookFee uint32
		chain   valueobject.ChainID
		native  bool // flag the first token as native, so Arc rescales it
	}{
		{"sample", "sample_pool.json", 0, valueobject.ChainIDBSC, false},
		{"thanos-h0", "testdata/thanos_pool.json", 0, valueobject.ChainIDBSC, false},
		{"thanos-h777", "testdata/thanos_pool.json", 777, valueobject.ChainIDBSC, false},
		{"thanos-h10000", "testdata/thanos_pool.json", 10000, valueobject.ChainIDBSC, false},
		{"thanos-h1000000", "testdata/thanos_pool.json", 1000000, valueobject.ChainIDBSC, false},
		{"sample-arc-native", "sample_pool.json", 0, valueobject.ChainIDArc, true},
		{"thanos-arc-native-h777", "testdata/thanos_pool.json", 777, valueobject.ChainIDArc, true},
	}
	amounts := []string{"0", "1", "1000", "1000000000000", "1000000000000000000", "3000000000000000000000",
		"900000000000000000000000000", new(big.Int).Lsh(big.NewInt(1), 127).String(),
		new(big.Int).Lsh(big.NewInt(1), 128).String(), new(big.Int).Lsh(big.NewInt(1), 255).String(),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(v.file)
			require.NoError(t, err)
			var ent entity.Pool
			require.NoError(t, json.Unmarshal(data, &ent))
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(ent.Extra), &extra))
			extra.HookSwapFee = v.hookFee
			eb, _ := json.Marshal(extra)
			ent.Extra = string(eb)
			base, err := NewPoolSimulator(ent, v.chain)
			require.NoError(t, err)
			if v.native {
				base.isNative[0] = true
			}
			successes := 0
			for dir := range 2 {
				for _, amt := range amounts {
					successes += sequentialBinParity(t, base, ent.Tokens[dir].Address, ent.Tokens[1-dir].Address, amt)
				}
			}
			if v.hookFee < 1000000 { // a 100% hook fee leaves nothing to swap, so every quote errors
				require.Positive(t, successes, "no quote succeeded; the parity check proves nothing")
			}
		})
	}
}

func sequentialBinParity(t *testing.T, base *PoolSimulator, tokenIn, tokenOut, amt string) int {
	t.Helper()
	nativeSim := base.CloneState().(*PoolSimulator)
	legacySim := base.CloneState().(*PoolSimulator)
	amtBig, _ := new(big.Int).SetString(amt, 10)
	amtU := uint256.MustFromBig(amtBig)
	successes := 0
	for step := range 3 {
		name := fmt.Sprintf("%s amt=%s step=%d", tokenIn, amt, step)
		lRes, lErr := legacySim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amtBig)}, TokenOut: tokenOut})
		nRes, nErr := nativeSim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU}, TokenOut: tokenOut})
		if lErr != nil || nErr != nil {
			require.Equal(t, fmt.Sprint(lErr), fmt.Sprint(nErr), name+" error")
			return successes
		}
		successes++
		require.Equal(t, lRes.TokenAmountOut.Amount.String(), nRes.TokenAmountOut.Amount.Dec(), name+" out")
		require.Equal(t, lRes.TokenAmountOut.Token, nRes.TokenAmountOut.Token, name+" out token")
		require.Equal(t, lRes.RemainingTokenAmountIn.Amount.String(), nRes.RemainingTokenAmountIn.Amount.Dec(), name+" remaining")
		require.Equal(t, lRes.RemainingTokenAmountIn.Token, nRes.RemainingTokenAmountIn.Token, name+" remaining token")
		require.Equal(t, lRes.Gas, nRes.Gas, name+" gas")
		require.Equal(t, lRes.SwapInfo, nRes.SwapInfo, name+" swapinfo")

		// exact-out on the same amount, then advance state: native UpdateState vs legacy UpdateBalance.
		lIn, lInErr := legacySim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amtBig)}, TokenIn: tokenOut})
		nIn, nInErr := nativeSim.CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: tokenIn, Amount: *amtU}, TokenIn: tokenOut})
		if lInErr != nil || nInErr != nil {
			require.Equal(t, fmt.Sprint(lInErr), fmt.Sprint(nInErr), name+" in error")
		} else {
			require.Equal(t, lIn.TokenAmountIn.Amount.String(), nIn.TokenAmountIn.Amount.Dec(), name+" in")
			require.Equal(t, lIn.TokenAmountIn.Token, nIn.TokenAmountIn.Token, name+" in token")
			require.Equal(t, lIn.RemainingTokenAmountOut.Amount.String(), nIn.RemainingTokenAmountOut.Amount.Dec(), name+" in remaining")
			require.Equal(t, lIn.Gas, nIn.Gas, name+" in gas")
			require.Equal(t, lIn.SwapInfo, nIn.SwapInfo, name+" in swapinfo")
			successes++
		}

		nativeSim.UpdateState(pool.UpdateStateParams{SwapInfo: nRes.SwapInfo})
		legacySim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: lRes.SwapInfo})
		require.Equal(t, legacySim.bins, nativeSim.bins, name+" bins")
		require.Equal(t, legacySim.activeId, nativeSim.activeId, name+" activeId")
	}
	return successes
}
