package hooks

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhitelistHook_GetFeeReturnsStaticFeeSentinel(t *testing.T) {
	h := NewWhitelistHook(nil)

	fee, err := h.GetFee(&GetFeeParams{Asset0IsInput: true})
	require.NoError(t, err)
	require.Equal(t, FeeUseStaticFee, fee)
	assert.Equal(t, uint64(math.MaxUint64), fee)
}

func TestWhitelistHook_BeforeSwap(t *testing.T) {
	enabled := NewWhitelistHook(nil).(*WhitelistHook)
	require.True(t, enabled.Extra.Enabled)
	assert.ErrorIs(t, enabled.BeforeSwap(&BeforeSwapParams{}), ErrSwapperNotAuthorized)

	disabled := NewWhitelistHook(&HookParam{HookExtra: `{"e":false}`}).(*WhitelistHook)
	require.False(t, disabled.Extra.Enabled)
	assert.ErrorIs(t, disabled.BeforeSwap(&BeforeSwapParams{}), ErrPoolDisabled)

	broken := NewWhitelistHook(&HookParam{HookExtra: `not-json`}).(*WhitelistHook)
	require.True(t, broken.Extra.Enabled)
}

func TestWhitelistHook_AfterSwapNoop(t *testing.T) {
	h := NewWhitelistHook(nil).(*WhitelistHook)
	assert.NoError(t, h.AfterSwap(&AfterSwapParams{}))
}

func TestWhitelistHook_FactoryRegistered(t *testing.T) {
	require.NotEmpty(t, WhitelistHookAddresses)

	for _, addr := range WhitelistHookAddresses {
		hook := GetHook(addr, &HookParam{HookExtra: `{"e":false}`})
		wl, ok := hook.(*WhitelistHook)
		require.True(t, ok)
		assert.False(t, wl.Extra.Enabled)

		assert.ErrorIs(t, hook.BeforeSwap(&BeforeSwapParams{}), ErrPoolDisabled)
	}
}

func TestWhitelistHook_UnknownAddressFallsBackToBase(t *testing.T) {
	hook := GetHook(common.HexToAddress("0x0000000000000000000000000000000000000001"), &HookParam{})
	_, ok := hook.(*BaseHook)
	assert.True(t, ok)
}

func TestWhitelistHook_CloneStateIsolation(t *testing.T) {
	h := NewWhitelistHook(&HookParam{HookExtra: `{"e":true}`}).(*WhitelistHook)

	cloned := h.CloneState().(*WhitelistHook)
	cloned.Extra.Enabled = false

	assert.True(t, h.Extra.Enabled)
	assert.ErrorIs(t, cloned.BeforeSwap(&BeforeSwapParams{}), ErrPoolDisabled)
}
