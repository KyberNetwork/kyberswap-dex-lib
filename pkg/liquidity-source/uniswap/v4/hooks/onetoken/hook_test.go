package onetoken

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// reference values are the live globals of the Base OneToken hook plus a real launch
// (WEREY, poolId 0xe23622f7d41653c012cfb72e9d4addbffcfd67f7fb142b998cbd050a861d94fc),
// cross-checked with cast on 2026-09-09, so the fee schedule is checked against the
// actual on-chain contract, not just its own logic restated.
func refExtra() Extra {
	return Extra{
		FeeBps:      300,        // 3%
		SnipeTaxBps: 1500,       // 15%
		SnipeWindow: 900,        // 15 min
		LaunchTime:  1788896153, // WEREY on Base
		OverrideBps: 0,
	}
}

// EffectiveFeeBps must reproduce the hook's effectiveFeeBps() exactly: a wrong bps
// here silently mis-prices every swap during the snipe window without erroring, so a
// route built on this quote settles for less than it should on-chain. Unlike b20's
// linear decay, ours is a step: the elevated tax for the whole window, then base.
func TestEffectiveFeeBps_Step(t *testing.T) {
	e := refExtra()

	cases := []struct {
		name    string
		elapsed int64
		want    int64
	}{
		{"at launch instant, snipe tax applies", 0, 1500},
		{"last second of the window, still snipe tax", 899, 1500},
		{"window boundary (elapsed == window), drops to base", 900, 300},
		{"long after the window, stays at base", 1_000_000, 300},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			NowFn = func() int64 { return e.LaunchTime + c.elapsed }
			assert.Equal(t, c.want, e.EffectiveFeeBps())
		})
	}
}

// Per-pool override applies only after the snipe window; during the window the snipe
// tax wins (matches effectiveFeeBps()'s priority: snipe > override > base).
func TestEffectiveFeeBps_Override(t *testing.T) {
	e := refExtra()
	e.OverrideBps = 500 // 5%

	NowFn = func() int64 { return e.LaunchTime } // in window
	assert.Equal(t, int64(1500), e.EffectiveFeeBps(), "snipe tax outranks override during the window")

	NowFn = func() int64 { return e.LaunchTime + e.SnipeWindow } // out of window
	assert.Equal(t, int64(500), e.EffectiveFeeBps(), "override applies after the window")
}

// The clamp must hold defensively: even an out-of-policy value that somehow made it
// through Track must never let the simulator quote a fee above the contract's hard
// ceiling (MAX_FEE_BPS = 20%). Use 9000 so the assertion can fail if the clamp is removed.
func TestEffectiveFeeBps_CappedAtMax(t *testing.T) {
	e := Extra{FeeBps: 9000, SnipeTaxBps: 9000, SnipeWindow: 10, LaunchTime: 1}

	NowFn = func() int64 { return 1 } // in window -> snipe tax path
	assert.Equal(t, int64(MaxFeeBps), e.EffectiveFeeBps())

	NowFn = func() int64 { return 1_000_000 } // out of window -> base path
	assert.Equal(t, int64(MaxFeeBps), e.EffectiveFeeBps())
}

// BeforeSwap sets the LP fee via SwapFee (pips = bps * 100), no delta, and is
// direction/exactness agnostic (a fee override is symmetric), so exact-in and
// exact-out return the same fee.
func TestBeforeSwap_SetsSwapFee(t *testing.T) {
	e := refExtra()
	NowFn = func() int64 { return e.LaunchTime + e.SnipeWindow } // post-window: 300 bps
	h := &Hook{Extra: e}

	for _, calcOut := range []bool{true, false} {
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: true})
		require.NoError(t, err)
		assert.Equal(t, uniswapv4.FeeAmount(300*feePipsPerBps), res.SwapFee) // 30_000 pips = 3%
		assert.Equal(t, bignumber.ZeroBI, res.DeltaSpecified)
		assert.Equal(t, bignumber.ZeroBI, res.DeltaUnspecified)
	}
}

// During the snipe window BeforeSwap must return the elevated fee.
func TestBeforeSwap_SnipeWindow(t *testing.T) {
	e := refExtra()
	NowFn = func() int64 { return e.LaunchTime } // elapsed 0: in window
	h := &Hook{Extra: e}

	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(1500*feePipsPerBps), res.SwapFee) // 150_000 pips = 15%
}

// A pool whose Track() never succeeded (RPC failure, or the Hook constructed before
// HookExtra is populated) has LaunchTime == 0 and must refuse to quote rather than
// default to a zero-value Extra (FeeBps == 0), which would silently price it fee-free.
func TestBeforeSwap_UntrackedPool_Errors(t *testing.T) {
	h := &Hook{} // zero-value Extra: LaunchTime == 0
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true})
	assert.ErrorIs(t, err, ErrNotLaunched)
}
