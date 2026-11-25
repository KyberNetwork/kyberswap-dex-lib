package testutil

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtOutParity checks that the native uint256 CalcAmtOut and the legacy big.Int CalcAmountOut agree on
// every token pair for each amount in amountsIn (decimal strings): same error text, amountOut, remaining amount
// and gas. A nil limit runs without a swap limit. Remove it with the legacy methods and the golden data.
func TestCalcAmtOutParity(t *testing.T, sim any, amountsIn []string, limit pool.SwapLimit) {
	t.Helper()
	legacy, ok := sim.(pool.ILegacyCalcAmountOut)
	require.True(t, ok, "%T has no legacy CalcAmountOut", sim)
	native, ok := sim.(pool.ICalcAmtOut)
	require.True(t, ok, "%T has no native CalcAmtOut", sim)

	successes := 0
	tokens := sim.(interface{ GetTokens() []string }).GetTokens()
	for i, tokenIn := range tokens {
		for j, tokenOut := range tokens {
			if i == j {
				continue
			}
			for _, amt := range amountsIn {
				name := fmt.Sprintf("%d->%d amt=%s", i, j, amt)
				amtBig, okBig := new(big.Int).SetString(amt, 10)
				require.True(t, okBig, name)
				lRes, lErr := safe(func() (*pool.CalcAmountOutResult, error) {
					return legacy.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amtBig)},
						TokenOut:      tokenOut,
						Limit:         pool.AsSwapLimitBig(limit),
					})
				})
				nRes, nErr := safe(func() (*pool.CalcAmountOutResult, error) {
					r, err := native.CalcAmtOut(pool.CalcAmtOutParams{
						TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *uint256.MustFromBig(amtBig)},
						TokenOut:      tokenOut,
						Limit:         limit,
					})
					if r == nil {
						return nil, err
					}
					return &pool.CalcAmountOutResult{
						TokenAmountOut:         bigAmt(r.TokenAmountOut),
						RemainingTokenAmountIn: bigAmt(r.RemainingTokenAmountIn),
						Gas:                    r.Gas,
					}, err
				})
				require.Equal(t, errText(lErr), errText(nErr), "%s error", name)
				if lErr != nil || lRes == nil || nRes == nil {
					require.Equal(t, lRes == nil, nRes == nil, "%s nil result", name)
					continue
				}
				successes++
				require.Equal(t, amount(lRes.TokenAmountOut), amount(nRes.TokenAmountOut), "%s amountOut", name)
				require.Equal(t, amount(lRes.RemainingTokenAmountIn), amount(nRes.RemainingTokenAmountIn), "%s remaining", name)
				require.Equal(t, lRes.Gas, nRes.Gas, "%s gas", name)
			}
		}
	}
	require.Positive(t, successes, "no amount quoted successfully; the parity check proves nothing")
}

func safe(f func() (*pool.CalcAmountOutResult, error)) (res *pool.CalcAmountOutResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("panic: %v", r)
		}
	}()
	return f()
}

func bigAmt(t *pool.TokenAmt) *pool.TokenAmount {
	if t == nil {
		return nil
	}
	return &pool.TokenAmount{Token: t.Token, Amount: t.Amount.ToBig()}
}

func amount(t *pool.TokenAmount) string {
	if t == nil || t.Amount == nil {
		return "0"
	}
	return t.Amount.String()
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// CalcAmountOutNative quotes with the native CalcAmtOut and returns big.Int amounts, or with the legacy
// CalcAmountOut for a pool that has no native method yet. For tests of a pool whose legacy method is gone.
func CalcAmountOutNative(sim any, in pool.TokenAmount, tokenOut string,
	limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	native, ok := sim.(pool.ICalcAmtOut)
	if !ok {
		return sim.(pool.ILegacyCalcAmountOut).CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: in, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(limit)})
	}
	res, err := native.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: in.Token, Amount: *uint256.MustFromBig(in.Amount)},
		TokenOut:      tokenOut,
		Limit:         limit,
	})
	if err != nil || res == nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         bigAmt(res.TokenAmountOut),
		Fee:                    &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: new(big.Int)}, // native results have no fee
		RemainingTokenAmountIn: bigAmt(res.RemainingTokenAmountIn),
		Gas:                    res.Gas,
		SwapInfo:               res.SwapInfo,
	}, nil
}

// TestUpdateStateParity checks that a legacy pool updated through the IPoolSim adapter (CalcAmtOut, then
// UpdateState without a fee) reaches the same state as CalcAmountOut then UpdateBalance with the fee set: same
// pool state, reserves and next quote. mk builds a fresh pool for each run.
func TestUpdateStateParity(t *testing.T, mk func() any, amountsIn []string) {
	t.Helper()
	// GetTokens plus every token reachable from them (a pool may swap to an LP token outside GetTokens).
	sim := mk().(interface {
		GetTokens() []string
		CanSwapFrom(string) []string
	})
	tokens := slices.Clone(sim.GetTokens())
	for _, tk := range sim.GetTokens() {
		for _, out := range sim.CanSwapFrom(tk) {
			if !slices.Contains(tokens, out) {
				tokens = append(tokens, out)
			}
		}
	}
	runs := 0
	for i, tokenIn := range tokens {
		for j, tokenOut := range tokens {
			if i == j {
				continue
			}
			for _, amt := range amountsIn {
				in := pool.TokenAmount{Token: tokenIn, Amount: bigFromDecimal(t, amt)}
				legacy := mk().(pool.IPoolSimulator)
				adaptedSim := mk()
				adapted := pool.AsIPoolSim(adaptedSim)
				lres, err := legacy.(pool.ILegacyCalcAmountOut).CalcAmountOut(
					pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: tokenOut})
				if err != nil {
					continue
				}
				inU := pool.TokenAmt{Token: tokenIn, Amount: *uint256.MustFromBig(in.Amount)}
				ares, err := adapted.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: inU, TokenOut: tokenOut})
				require.NoError(t, err)
				legacy.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *lres.TokenAmountOut,
					Fee: *lres.Fee, SwapInfo: lres.SwapInfo})
				adapted.UpdateState(pool.UpdateStateParams{TokenAmountIn: inU, TokenAmountOut: *ares.TokenAmountOut,
					SwapInfo: ares.SwapInfo})
				runs++
				label := fmt.Sprintf("%s -> %s amountIn %s", tokenIn, tokenOut, amt)
				require.Equal(t, legacy, adaptedSim, "pool state, %s", label)
				lr, ar := legacy.GetReserves(), adapted.GetReserves()
				require.Len(t, ar, len(lr), label)
				for k := range lr {
					require.Equal(t, lr[k].String(), ar[k].String(), "reserve %d, %s", k, label)
				}
				lnext, lerr := legacy.(pool.ILegacyCalcAmountOut).CalcAmountOut(
					pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: tokenOut})
				anext, aerr := adapted.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: inU, TokenOut: tokenOut})
				require.Equal(t, lerr == nil, aerr == nil, "next quote error, %s", label)
				if lerr == nil {
					require.Equal(t, lnext.TokenAmountOut.Amount.String(), anext.TokenAmountOut.Amount.String(),
						"next quote, %s", label)
				}
			}
		}
	}
	require.NotZero(t, runs, "no successful swap to compare")
}

func bigFromDecimal(t testing.TB, s string) *big.Int {
	t.Helper()
	b, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, s)
	return b
}
