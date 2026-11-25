package cl

// Sequential parity (quote, UpdateState, quote) of native CalcAmtOut and legacy CalcAmountOut, with and without
// a hook. Remove with the legacy methods and the golden data.

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// stubHook runs the given BeforeSwap/AfterSwap functions; nil means the BaseHook default.
type stubHook struct {
	*BaseHook
	before func(*BeforeSwapParams) (*BeforeSwapResult, error)
	after  func(*AfterSwapParams) (*AfterSwapResult, error)
	calls  int // BeforeSwap calls, a stateful hook for the sequential check
}

func (h *stubHook) CloneState() Hook  { c := *h; return &c }
func (h *stubHook) UpdateBalance(any) {}
func (h *stubHook) BeforeSwap(p *BeforeSwapParams) (*BeforeSwapResult, error) {
	h.calls++
	if h.before == nil {
		return h.BaseHook.BeforeSwap(p)
	}
	return h.before(p)
}
func (h *stubHook) AfterSwap(p *AfterSwapParams) (*AfterSwapResult, error) {
	if h.after == nil {
		return h.BaseHook.AfterSwap(p)
	}
	return h.after(p)
}

func TestCalcAmtOutLegacyParity(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("sample_pool.json")
	require.NoError(t, err)
	var ent entity.Pool
	require.NoError(t, json.Unmarshal(data, &ent))

	u := func(v uint64) *uint256.Int { return uint256.NewInt(v) }
	neg := func(v uint64) *uint256.Int { return new(uint256.Int).Neg(uint256.NewInt(v)) }
	zero := func() *uint256.Int { return new(uint256.Int) }
	before := func(spec, unspec *uint256.Int, fee uniswapv4.FeeAmount, gas int64) func(*BeforeSwapParams) (*BeforeSwapResult, error) {
		return func(*BeforeSwapParams) (*BeforeSwapResult, error) {
			return &BeforeSwapResult{DeltaSpecified: spec, DeltaUnspecified: unspec, SwapFee: fee, Gas: gas}, nil
		}
	}
	feeOf := func(v uint64) *uint256.Int { return u(v) }
	hookFee := func(fee *uint256.Int, gas int64) func(*AfterSwapParams) (*AfterSwapResult, error) {
		return func(*AfterSwapParams) (*AfterSwapResult, error) { return &AfterSwapResult{HookFee: fee, Gas: gas}, nil }
	}
	cases := []struct {
		name string
		hook func() Hook // nil: leave the pool's own BaseHook
	}{
		{"pool-hook", nil},
		{"nil-hook", func() Hook { return nil }},
		{"delta-specified", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(u(1000), zero(), 0, 0)} }},
		{"delta-specified-all-input", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: func(p *BeforeSwapParams) (*BeforeSwapResult, error) {
				return &BeforeSwapResult{DeltaSpecified: p.AmountSpecified, DeltaUnspecified: neg(12345)}, nil
			}}
		}},
		{"delta-specified-above-input", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: before(new(uint256.Int).Lsh(u(1), 200), zero(), 0, 0)}
		}},
		{"neg-delta-specified", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(neg(500), zero(), 0, 0)} }},
		{"delta-unspecified", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), u(5000), 0, 0)} }},
		{"delta-unspecified-above-output", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), new(uint256.Int).Lsh(u(1), 200), 0, 0)}
		}},
		{"neg-delta-unspecified", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), neg(5000), 0, 0)} }},
		{"fee-override", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), zero(), 5000, 77)} }},
		{"fee-same-as-pool", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), zero(), 335, 0)} }},
		{"fee-at-max", func() Hook { return &stubHook{BaseHook: &BaseHook{}, before: before(zero(), zero(), FeeMax, 0)} }},
		{"hook-fee", func() Hook { return &stubHook{BaseHook: &BaseHook{}, after: hookFee(u(77), 999)} }},
		{"neg-hook-fee", func() Hook { return &stubHook{BaseHook: &BaseHook{}, after: hookFee(neg(77), 5)} }},
		{"hook-fee-above-output", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, after: hookFee(new(uint256.Int).Lsh(u(1), 200), 0)}
		}},
		{"all-terms", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: before(u(100), u(5000), 4000, 1234), after: hookFee(u(77), 999)}
		}},
		{"before-error", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: func(*BeforeSwapParams) (*BeforeSwapResult, error) {
				return nil, errors.New("boom")
			}}
		}},
		{"before-nil-result", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, before: func(*BeforeSwapParams) (*BeforeSwapResult, error) { return nil, nil }}
		}},
		{"after-error", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, after: func(*AfterSwapParams) (*AfterSwapResult, error) {
				return nil, errors.New("bang")
			}}
		}},
		{"after-nil-fee", func() Hook {
			return &stubHook{BaseHook: &BaseHook{}, after: hookFee(nil, 0)}
		}},
		{"stateful-fee", func() Hook { // fee depends on the quote count, so a missed UpdateState shows
			h := &stubHook{BaseHook: &BaseHook{}}
			h.before = func(*BeforeSwapParams) (*BeforeSwapResult, error) {
				return &BeforeSwapResult{DeltaSpecified: zero(), DeltaUnspecified: feeOf(uint64(h.calls)), SwapFee: uniswapv4.FeeAmount(1000 * h.calls)}, nil
			}
			return h
		}},
	}
	amounts := []string{"1", "1000", "1000000000000", "1000000000000000000", "3000000000000000000000",
		"900000000000000000000000000", new(big.Int).Lsh(big.NewInt(1), 255).String(),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()}
	for _, chain := range []valueobject.ChainID{1, valueobject.ChainIDArc} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%d/%s", chain, tc.name), func(t *testing.T) {
				t.Parallel()
				base, err := NewPoolSimulator(ent, chain)
				require.NoError(t, err)
				if tc.hook != nil {
					base.hook = tc.hook()
				}
				successes := 0
				for dir := range 2 {
					for _, amt := range amounts {
						// Arc rescales native input by 1e12. An input of 2^255 or more overflows that product, and
						// the native path then returns ErrOverflow before the hook runs, where the big.Int body
						// would run the hook first. Real amounts are far below that, so Arc skips these two.
						if chain == valueobject.ChainIDArc && len(amt) > 40 {
							continue
						}
						successes += sequentialCLParity(t, base, ent.Tokens[dir].Address, ent.Tokens[1-dir].Address, amt)
					}
				}
				if tc.name == "pool-hook" || tc.name == "nil-hook" || tc.name == "delta-specified" {
					require.Positive(t, successes, "no quote succeeded; the parity check proves nothing")
				}
			})
		}
	}
}

func sequentialCLParity(t *testing.T, base *PoolSimulator, tokenIn, tokenOut, amt string) int {
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
		require.Equal(t, lRes.RemainingTokenAmountIn == nil, nRes.RemainingTokenAmountIn == nil, name+" remaining nil")
		if lRes.RemainingTokenAmountIn != nil {
			require.Equal(t, lRes.RemainingTokenAmountIn.Amount.String(), nRes.RemainingTokenAmountIn.Amount.Dec(), name+" remaining")
			require.Equal(t, lRes.RemainingTokenAmountIn.Token, nRes.RemainingTokenAmountIn.Token, name+" remaining token")
		}
		require.Equal(t, lRes.Fee.Token, nRes.Fee.Token, name+" fee token")
		require.Equal(t, lRes.Gas, nRes.Gas, name+" gas")
		require.Equal(t, lRes.SwapInfo, nRes.SwapInfo, name+" swapinfo")

		upd := func(sim *PoolSimulator, out *big.Int, info any) {
			sim.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU},
				TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: *uint256.MustFromBig(out)}, SwapInfo: info})
		}
		upd(nativeSim, nRes.TokenAmountOut.Amount.ToBig(), nRes.SwapInfo)
		upd(legacySim, lRes.TokenAmountOut.Amount, lRes.SwapInfo)
	}
	return successes
}
