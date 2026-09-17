package coocoo

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Every CooCoo hook address resolves to this package's handler under its own exchange, with
// the tracked Extra restored from HookExtra. The addresses' permission bits (…2044, …E044)
// enable afterSwap and afterSwapReturnDelta and never beforeSwap, like Pons' own hook.
func TestRegistered(t *testing.T) {
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{
			HookExtra: uniswapv4.HookExtra(`{"r":true,"f":150}`),
		})
		require.True(t, ok, addr)
		h, isCooCoo := hook.(*Hook)
		require.True(t, isCooCoo, addr)
		assert.Equal(t, string(valueobject.ExchangeUniswapV4CooCoo), h.GetExchange())
		assert.True(t, h.Registered)
		assert.Equal(t, int64(150), h.FeeBps)
		assert.True(t, uniswapv4.HasSwapPermissions(addr))
		assert.True(t, h.CanAfterSwap(addr))
		assert.False(t, h.CanBeforeSwap(addr))
	}
}

// Same swap, same fee as hooks/pons-v2 on both sides: the deployments share
// PonsV2MemeHook's bytecode, so the fee is floor(unspecified * feeBps / 10_000) of the
// realized output on exact-in and of the realized input on exact-out.
func TestAfterSwap_MatchesPonsV2(t *testing.T) {
	ours := &Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 150}}
	theirs := &ponsv2.Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 150}}
	for _, calcOut := range []bool{true, false} {
		params := &uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: calcOut},
			AmountIn:         big.NewInt(1_000_000),
			AmountOut:        big.NewInt(2_000_000),
		}
		got, err := ours.AfterSwap(params)
		require.NoError(t, err)
		want, err := theirs.AfterSwap(params)
		require.NoError(t, err)
		assert.Equal(t, want.HookFee, got.HookFee)
		// exact-in: floor(2_000_000 * 150 / 10_000); exact-out: floor(1_000_000 * 150 / 10_000)
		assert.Equal(t, big.NewInt(map[bool]int64{true: 30_000, false: 15_000}[calcOut]), got.HookFee)
	}
}

// A pool Track() never saw prices fee-free rather than erroring: PonsV2MemeHook._afterSwap
// itself returns 0 for an unregistered pool.
func TestAfterSwap_Unregistered_NoFee(t *testing.T) {
	result, err := (&Hook{}).AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

// What Track hands back is what the factory restores from HookExtra on the next load.
func TestExtraRoundTrip(t *testing.T) {
	h := &Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 100}}
	raw, err := json.Marshal(h)
	require.NoError(t, err)
	assert.JSONEq(t, `{"r":true,"f":100}`, string(raw))

	restored, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)})
	require.True(t, ok)
	assert.Equal(t, h.Extra, restored.(*Hook).Extra)
}
