package uniswapv4

import (
	"errors"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCalcAmtInParity: native CalcAmtIn must match the legacy CalcAmountIn on amountIn, gas and error. The golden
// amounts are the legacy outputs captured before the native path existed, so a drift in the shared body fails here.
func TestCalcAmtInParity(t *testing.T) {
	t.Parallel()
	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))
	base, err := NewPoolSimulator(poolEnt, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	dynamic := *base
	dynamic.staticExtra.Fee = 0x800000

	swappingHook := func() *stubHook {
		return &stubHook{BaseHook: &BaseHook{}, canBefore: true, before: &BeforeSwapResult{
			DeltaSpecified: uint256.NewInt(100), DeltaUnspecified: uint256.NewInt(5000), SwapFee: 5000, Gas: 1234,
		}}
	}
	const usdc, rez = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0x3b50805453023a91a8bf641e279401a0b23fa6f9"
	withHook := func(sim PoolSimulator) PoolSimulator { sim.hook = swappingHook(); return sim }
	cases := []struct {
		name    string
		sim     PoolSimulator
		tokenIn string
		amtOut  string
		wantIn  string // empty means the quote errors
		wantGas int64
	}{
		{"no-hook usdc-in", *base, usdc, "1000000", "2", 129869},
		{"no-hook usdc-in large", *base, usdc, "15497045801", "2", 129869},
		{"no-hook rez-in", *base, rez, "1000000", "78544796805914592071", 129869},
		{"no-hook rez-in gas", *base, rez, "15497045801", "1297993471066489845784479", 145329},
		{"no-hook usdc-in max", *base, usdc, "1070524829112273927801315", "15497045800", 129869},
		{"no-hook usdc-in 1e21", *base, usdc, "1000000000000000000000", "12991435", 129869},
		{"no-hook rez-in insufficient", *base, rez, "1070524829112273927801315", "", 0},
		{"no-hook rez-in insufficient 1e21", *base, rez, "1000000000000000000000", "", 0},
		{"hook usdc-in", withHook(*base), usdc, "1000000", "5002", 131103},
		{"hook usdc-in large", withHook(*base), usdc, "15497045801", "5002", 131103},
		{"hook rez-in", withHook(*base), rez, "1000000", "78552651344163611641", 131103},
		{"hook rez-in gas", withHook(*base), rez, "15497045801", "1297993479568861753129398", 146563},
		{"hook usdc-in max", withHook(*base), usdc, "1070524829112273927801315", "15497050802", 131103},
		{"hook usdc-in 1e21", withHook(*base), usdc, "1000000000000000000000", "12996435", 131103},
		{"hook rez-in insufficient", withHook(*base), rez, "1070524829112273927801315", "", 0},
		{"dynamic-fee hook rez-in", withHook(dynamic), rez, "1000000", "78157914402735653819", 131103},
		{"dynamic-fee hook rez-in gas", withHook(dynamic), rez, "15497045801", "1291470899269520739294602", 146563},
		{"dynamic-fee hook usdc-in max", withHook(dynamic), usdc, "1070524829112273927801315", "15419176199", 131103},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sim := tc.sim
			tokenOut := otherToken(tc.tokenIn)
			amtOut, ok := new(big.Int).SetString(tc.amtOut, 10)
			require.True(t, ok)
			var amtOutU256 uint256.Int
			require.False(t, amtOutU256.SetFromBig(amtOut))

			want, wantErr := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: amtOut},
				TokenIn:        tc.tokenIn,
			})
			got, gotErr := sim.CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: amtOutU256},
				TokenIn:        tc.tokenIn,
			})
			if tc.wantIn == "" {
				require.Error(t, gotErr)
				require.Error(t, wantErr)
				assert.Equal(t, wantErr.Error(), gotErr.Error(), "error")
				return
			}
			require.NoError(t, wantErr)
			require.NoError(t, gotErr)
			assert.Equal(t, tc.wantIn, got.TokenAmountIn.Amount.Dec(), "golden amountIn")
			assert.Equal(t, want.TokenAmountIn.Amount.String(), got.TokenAmountIn.Amount.Dec(), "amountIn")
			assert.Equal(t, tc.wantGas, got.Gas, "golden gas")
			assert.Equal(t, want.Gas, got.Gas, "gas")
		})
	}
}

// TestCalcAmtInSwappingHookError: a failing BeforeSwap surfaces through CalcAmtIn with the legacy error text.
func TestCalcAmtInSwappingHookError(t *testing.T) {
	t.Parallel()
	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))
	base, err := NewPoolSimulator(poolEnt, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	const usdc, rez = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0x3b50805453023a91a8bf641e279401a0b23fa6f9"
	base.hook = &stubHook{BaseHook: &BaseHook{}, canBefore: true, beforeErr: errors.New("boom")}

	_, err = base.CalcAmtIn(pool.CalcAmtInParams{
		TokenAmountOut: pool.TokenAmt{Token: rez, Amount: *uint256.NewInt(1000000)},
		TokenIn:        usdc,
	})
	require.Error(t, err)
	assert.Equal(t, "[BeforeSwap] boom", err.Error())
}

// otherToken returns the pool token that is not tokenIn, the output side of an exact-out quote.
func otherToken(tokenIn string) string {
	const usdc, rez = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0x3b50805453023a91a8bf641e279401a0b23fa6f9"
	if tokenIn == usdc {
		return rez
	}
	return usdc
}
