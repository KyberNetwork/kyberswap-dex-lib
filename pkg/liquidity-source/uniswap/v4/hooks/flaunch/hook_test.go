package flaunch

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func defaultHook() *Hook {
	return &Hook{Hook: &uniswapv4.BaseHook{}, Extra: Extra{SwapFee: DefaultSwapFee}}
}

func TestHook_BeforeSwap_TakesNoFee(t *testing.T) {
	for _, calcOut := range []bool{true, false} {
		result, err := defaultHook().BeforeSwap(&uniswapv4.BeforeSwapParams{AmountSpecified: big.NewInt(1e18), CalcOut: calcOut})
		require.NoError(t, err)
		assert.Equal(t, uniswapv4.FeeAmount(0), result.SwapFee)
		assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified)
		assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
	}
}

// The fee is floor(unspecified * swapFee / 1e4) on the unspecified side: the output for
// exact-in, the input for exact-out. Mirrors FeeDistributor._captureSwapFees.
func TestHook_AfterSwap_FeeOnUnspecified(t *testing.T) {
	cases := []struct {
		name    string
		swapFee uint32
		calcOut bool
		in, out int64
		want    int64
	}{
		{"exact-in, default 1% of output", DefaultSwapFee, true, 5e18, 1e18, 1e16},
		{"exact-out, default 1% of input", DefaultSwapFee, false, 1e18, 5e18, 1e16},
		{"exact-in, pool override 2.5%", 250, true, 1, 1e18, 25e15},
		{"exact-in, floor rounding", DefaultSwapFee, true, 1, 199, 1},
		{"exact-in, below one wei of fee rounds to zero", DefaultSwapFee, true, 1, 99, 0},
		{"fee-free pool override", 0, true, 1, 1e18, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := defaultHook()
			h.SwapFee = c.swapFee
			result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
				BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: c.calcOut},
				AmountIn:         big.NewInt(c.in),
				AmountOut:        big.NewInt(c.out),
			})
			require.NoError(t, err)
			assert.Zero(t, big.NewInt(c.want).Cmp(result.HookFee), "want %d got %s", c.want, result.HookFee)
		})
	}
}

func TestHook_AfterSwap_RejectsFeeAbove100Percent(t *testing.T) {
	h := defaultHook()
	h.SwapFee = FeeDenom + 1
	_, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountIn: big.NewInt(1), AmountOut: big.NewInt(1e18),
	})
	assert.ErrorIs(t, err, ErrFeeTooHigh)
}

// A pool behind an enforcing spend gate reverts on-chain for any swap that lacks a signed
// authorization, so both hook stages must refuse to quote it rather than mis-price it. The gate
// stops enforcing once endsAt has passed (block.timestamp > endsAt), or when it is disabled.
func TestHook_SpendGate(t *testing.T) {
	const now = 1_800_000_000
	NowFn = func() int64 { return now }
	t.Cleanup(func() { NowFn = func() int64 { return now } })

	swap := &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true}, AmountIn: big.NewInt(1), AmountOut: big.NewInt(1e18),
	}
	cases := []struct {
		name    string
		extra   Extra
		wantErr error
	}{
		{"gate enabled, no expiry", Extra{SwapFee: DefaultSwapFee, GateEnabled: true}, ErrGated},
		{"gate enabled, expires later", Extra{SwapFee: DefaultSwapFee, GateEnabled: true, GateEndsAt: now + 60}, ErrGated},
		{"gate enabled, expires exactly now (still enforcing)", Extra{SwapFee: DefaultSwapFee, GateEnabled: true, GateEndsAt: now}, ErrGated},
		{"gate enabled but expired", Extra{SwapFee: DefaultSwapFee, GateEnabled: true, GateEndsAt: now - 1}, nil},
		{"gate disabled with stale endsAt", Extra{SwapFee: DefaultSwapFee, GateEndsAt: now + 60}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := &Hook{Hook: &uniswapv4.BaseHook{}, Extra: c.extra}
			_, beforeErr := h.BeforeSwap(swap.BeforeSwapParams)
			_, afterErr := h.AfterSwap(swap)
			if c.wantErr != nil {
				assert.ErrorIs(t, beforeErr, c.wantErr)
				assert.ErrorIs(t, afterErr, c.wantErr)
				return
			}
			assert.NoError(t, beforeErr)
			assert.NoError(t, afterErr)
		})
	}
}

// Factory: a pool tracked before this change has no HookExtra (or a foreign schema) and must keep
// pricing at the protocol default until its next Track; our own schema is restored verbatim.
func TestFactory_HookExtra(t *testing.T) {
	factory := uniswapv4.HookFactories[HookAddresses[0]]
	require.NotNil(t, factory)

	fromEmpty := factory(&uniswapv4.HookParam{}).(*Hook)
	assert.Equal(t, Extra{SwapFee: DefaultSwapFee}, fromEmpty.Extra)

	fromForeign := factory(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"fee":"12345","model":"auto"}`)}).(*Hook)
	assert.Equal(t, Extra{SwapFee: DefaultSwapFee}, fromForeign.Extra)

	persisted, err := json.Marshal(&Hook{Extra: Extra{SwapFee: 250, GateEnabled: true, GateEndsAt: 1_800_000_000}})
	require.NoError(t, err)
	fromOwn := factory(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(persisted)}).(*Hook)
	assert.Equal(t, Extra{SwapFee: 250, GateEnabled: true, GateEndsAt: 1_800_000_000}, fromOwn.Extra)
}

func TestRegistry_AllAddressesRegistered(t *testing.T) {
	seen := map[string]bool{}
	for _, addr := range HookAddresses {
		assert.False(t, seen[addr.Hex()], "duplicate hook address %s", addr.Hex())
		seen[addr.Hex()] = true
		_, ok := uniswapv4.HookFactories[addr]
		assert.True(t, ok, "hook %s not registered", addr.Hex())
		assert.True(t, uniswapv4.HasSwapPermissions(addr), "hook %s lacks swap permissions", addr.Hex())
	}
}

func TestCloneState_IsIndependent(t *testing.T) {
	h := defaultHook()
	cloned := h.CloneState().(*Hook)
	cloned.SwapFee = 500
	assert.Equal(t, uint32(DefaultSwapFee), h.SwapFee)
}
