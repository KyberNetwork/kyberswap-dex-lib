package gluehook

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestHookRegistered(t *testing.T) {
	t.Parallel()
	// the V3 address (the only one registered) resolves to the passthrough hook
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, nil)
		require.True(t, ok, addr.Hex())
		assert.Equal(t, string(valueobject.ExchangeUniswapV4GlueHook), hook.GetExchange())
	}
}

// GlueHook never changes the swapper's amounts (no beforeSwap; the pump is the hook's own swap
// in afterSwap from the pot's balance) — the simulation must be a pure passthrough with only a
// gas budget on top.
func TestPassthroughQuoting(t *testing.T) {
	t.Parallel()
	hook, _ := uniswapv4.GetHook(HookAddresses[0], nil)

	for _, params := range []*uniswapv4.BeforeSwapParams{
		{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1e18)},
		{CalcOut: true, ZeroForOne: false, AmountSpecified: big.NewInt(1e18)},
		{CalcOut: false, ZeroForOne: true, AmountSpecified: big.NewInt(1e18)},
		{CalcOut: false, ZeroForOne: false, AmountSpecified: big.NewInt(1e18)},
	} {
		beforeSwapResult, err := hook.BeforeSwap(params)
		require.NoError(t, err)
		require.NoError(t, uniswapv4.ValidateBeforeSwapResult(beforeSwapResult))
		assert.Zero(t, beforeSwapResult.DeltaSpecified.Sign())
		assert.Zero(t, beforeSwapResult.DeltaUnspecified.Sign())
		assert.EqualValues(t, maxHookGas, beforeSwapResult.Gas)

		afterSwapResult, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: params,
			AmountIn:         big.NewInt(1e18),
			AmountOut:        big.NewInt(1e18),
		})
		require.NoError(t, err)
		require.NoError(t, uniswapv4.ValidateAfterSwapResult(afterSwapResult))
		assert.Zero(t, afterSwapResult.HookFee.Sign())
	}
}
