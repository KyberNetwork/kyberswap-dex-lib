package clanker

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCloneStateIsolatesPoolFVars pins the isolation contract of CloneState. PoolFVars is a
// pointer, so the struct copy inside CloneState left it shared with the source, while
// getVolatilityAccumulator writes ReferenceTick / ResetTick / AppliedVR / LastSwapTimestamp
// through it on every BeforeSwap — a clone therefore rewrote the source's fee state, and
// CloneState itself replaced the source's timestamps on the way out.
func TestCloneStateIsolatesPoolFVars(t *testing.T) {
	t.Parallel()

	hookExtra := `{"p":1000,"f":{"r":100,"t":200,"s":"1700000000","l":"1700000001","a":42,` +
		`"p":"7"},"c":{"b":3000,"m":30000}}`

	src, ok := NewDynamicFeeHook(Clanker)(&uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: valueobject.ChainIDBase},
		HookExtra: uniswapv4.HookExtra(hookExtra),
	}).(*DynamicFeeHook)
	require.True(t, ok)
	require.NotNil(t, src.PoolFVars, "fixture must carry PoolFVars, else this proves nothing")

	before := *src.PoolFVars

	cloned, ok := src.CloneState().(*DynamicFeeHook)
	require.True(t, ok)

	require.NotSame(t, src.PoolFVars, cloned.PoolFVars,
		"the clone must not share PoolFVars with the source")
	require.Same(t, before.ResetTickTimestamp, src.PoolFVars.ResetTickTimestamp,
		"CloneState must not rebind the source's own timestamps")
	require.Same(t, before.LastSwapTimestamp, src.PoolFVars.LastSwapTimestamp)

	// Everything getVolatilityAccumulator writes must land on the clone alone.
	cloned.PoolFVars.ReferenceTick = 999
	cloned.PoolFVars.ResetTick = 888
	cloned.PoolFVars.AppliedVR = 777
	cloned.PoolFVars.LastSwapTimestamp = uint256.NewInt(1)
	cloned.PoolFVars.ResetTickTimestamp = uint256.NewInt(2)

	require.Equal(t, before.ReferenceTick, src.PoolFVars.ReferenceTick)
	require.Equal(t, before.ResetTick, src.PoolFVars.ResetTick)
	require.Equal(t, before.AppliedVR, src.PoolFVars.AppliedVR)
	require.Equal(t, before.LastSwapTimestamp, src.PoolFVars.LastSwapTimestamp)
	require.Equal(t, before.ResetTickTimestamp, src.PoolFVars.ResetTickTimestamp)
}

// TestCloneStateNilPoolFVars: PoolFVars is omitempty, so a hook can legitimately arrive without it
// and CloneState must not panic dereferencing it.
func TestCloneStateNilPoolFVars(t *testing.T) {
	t.Parallel()

	src, ok := NewDynamicFeeHook(Clanker)(&uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: valueobject.ChainIDBase},
		HookExtra: uniswapv4.HookExtra(`{"p":1000}`),
	}).(*DynamicFeeHook)
	require.True(t, ok)
	require.Nil(t, src.PoolFVars)
	src.ProtocolFee = big.NewInt(1000)

	require.NotPanics(t, func() { _ = src.CloneState() })
}
