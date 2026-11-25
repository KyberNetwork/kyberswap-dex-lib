package testutil

import (
	"fmt"
	"math/big"
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
		Fee:                    bigAmt(res.Fee),
		RemainingTokenAmountIn: bigAmt(res.RemainingTokenAmountIn),
		Gas:                    res.Gas,
		SwapInfo:               res.SwapInfo,
	}, nil
}
