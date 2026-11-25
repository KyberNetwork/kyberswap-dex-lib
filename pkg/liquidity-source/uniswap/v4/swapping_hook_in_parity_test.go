package uniswapv4

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCalcAmtInHookTermsParity: native CalcAmtIn must match the legacy CalcAmountIn to the wei (amountIn, remaining,
// tokens, gas, SwapInfo, error text) for every swapping-hook term, signed hook deltas, a dynamic swap fee and
// outputs near and above the pool reserves, in both directions.
func TestCalcAmtInHookTermsParity(t *testing.T) {
	t.Parallel()
	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))
	base, err := NewPoolSimulator(poolEnt, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	dynamic := *base
	dynamic.staticExtra.Fee = 0x800000

	u := func(v uint64) *uint256.Int { return uint256.NewInt(v) }
	neg := func(v uint64) *uint256.Int { return new(uint256.Int).Neg(uint256.NewInt(v)) }
	huge := new(uint256.Int).Lsh(u(1), 200)
	maxInt := new(uint256.Int).Sub(new(uint256.Int).Lsh(u(1), 255), u(1))
	before := func(spec, unspec *uint256.Int, fee FeeAmount, gas int64) *stubHook {
		return &stubHook{BaseHook: &BaseHook{}, canBefore: true,
			before: &BeforeSwapResult{DeltaSpecified: spec, DeltaUnspecified: unspec, SwapFee: fee, Gas: gas}}
	}
	after := func(fee *uint256.Int, gas int64) *stubHook {
		return &stubHook{BaseHook: &BaseHook{}, canAfter: true, after: &AfterSwapResult{HookFee: fee, Gas: gas}}
	}
	both := func(b *stubHook, a *stubHook) *stubHook { b.canAfter, b.after = true, a.after; return b }
	cases := []struct {
		name string
		sim  PoolSimulator
		hook func() *stubHook
	}{
		{"no-hook", *base, func() *stubHook { return nil }},
		{"inert-hook", *base, func() *stubHook { return &stubHook{BaseHook: &BaseHook{}} }},
		{"delta-specified", *base, func() *stubHook { return before(u(100), u(0), 0, 0) }},
		{"delta-unspecified", *base, func() *stubHook { return before(u(0), u(5000), 0, 0) }},
		{"swap-fee", dynamic, func() *stubHook { return before(u(0), u(0), 5000, 0) }},
		{"fee-at-max", dynamic, func() *stubHook { return before(u(0), u(0), FeeMax, 0) }},
		{"before-gas", *base, func() *stubHook { return before(u(0), u(0), 0, 1234) }},
		{"after-hook-fee", *base, func() *stubHook { return after(u(77), 999) }},
		{"all-terms", dynamic, func() *stubHook {
			return both(before(u(100), u(5000), 5000, 1234), after(u(77), 999))
		}},
		{"before-error", *base, func() *stubHook {
			return &stubHook{BaseHook: &BaseHook{}, canBefore: true, beforeErr: fmt.Errorf("boom")}
		}},
		{"neg-delta-specified", *base, func() *stubHook { return before(neg(100), u(0), 0, 0) }},
		{"neg-delta-specified-below-zero", *base, func() *stubHook { return before(neg(1_000_000_000), u(0), 0, 0) }},
		{"neg-delta-unspecified", *base, func() *stubHook { return before(u(0), neg(5000), 0, 0) }},
		{"neg-hook-fee", *base, func() *stubHook { return after(neg(77), 0) }},
		{"delta-specified-huge", *base, func() *stubHook { return before(huge, u(0), 0, 0) }},
		{"delta-specified-max-int", *base, func() *stubHook { return before(maxInt, u(0), 0, 0) }},
		{"delta-unspecified-huge", *base, func() *stubHook { return before(u(0), huge, 0, 0) }},
		{"unspecified-negative-then-fee-refund", *base, func() *stubHook {
			return both(before(u(0), neg(1_000_000), 0, 0), after(u(2_000_000), 0))
		}},
	}
	const usdc, rez = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0x3b50805453023a91a8bf641e279401a0b23fa6f9"
	amounts := []string{"0", "1", "1000000", "15497045801", "1000000000000000000000",
		"15497045800", "1070524829112273927801315", "1070524829112273927801316",
		new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(10)).String(),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			successes := 0
			for _, tokenIn := range []string{usdc, rez} {
				tokenOut := otherToken(tokenIn)
				for _, amt := range amounts {
					sim := tc.sim
					if h := tc.hook(); h != nil {
						sim.hook = h
					}
					amtBig, _ := new(big.Int).SetString(amt, 10)
					name := fmt.Sprintf("%s in=%s out=%s", tc.name, tokenIn[:6], amt)
					want, wantErr := sim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: new(big.Int).Set(amtBig)}, TokenIn: tokenIn})
					got, gotErr := sim.CalcAmtIn(pool.CalcAmtInParams{
						TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: *uint256.MustFromBig(amtBig)}, TokenIn: tokenIn})
					if wantErr != nil || gotErr != nil {
						require.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr), name+" error")
						require.Nil(t, got, name)
						continue
					}
					successes++
					require.Equal(t, want.TokenAmountIn.Amount.String(), got.TokenAmountIn.Amount.Dec(), name+" in")
					require.Equal(t, want.TokenAmountIn.Token, got.TokenAmountIn.Token, name+" in token")
					require.Equal(t, want.RemainingTokenAmountOut.Amount.String(),
						got.RemainingTokenAmountOut.Amount.Dec(), name+" remaining")
					require.Equal(t, want.RemainingTokenAmountOut.Token, got.RemainingTokenAmountOut.Token, name+" remaining token")
					require.Equal(t, want.Gas, got.Gas, name+" gas")
					require.Equal(t, want.SwapInfo, got.SwapInfo, name+" swapinfo")
				}
			}
			t.Logf("%s: %d successes", tc.name, successes)
		})
	}
}
