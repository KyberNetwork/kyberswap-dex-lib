package o1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/b20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// reference values from a real o1 launch on Base (poolId
// 0xd980e584...92da0e, via `cast call poolConfig(bytes32)` on 2026-09-06).
// Exercises b20.Extra's shared logic through o1's embedding; decay/fee math
// itself is already covered by hooks/b20/hook_test.go.
func refExtra() b20.Extra {
	return b20.Extra{
		TokenIsCurrency0:       false,
		BaseFeeBps:             100,
		AntiSnipeStartTotalBps: 9900,
		AntiSnipeWindowSeconds: 20,
		LaunchTime:             1788228447,
	}
}

func TestTotalFeeBps_Decay(t *testing.T) {
	e := refExtra()

	cases := []struct {
		name    string
		elapsed int64
		want    int64
	}{
		{"at launch instant, surcharge is maximal", 0, 9900},
		{"halfway through the window, surcharge is halved", 10, 5000},
		{"window boundary, decays to just base fee", 20, 100},
		{"long after the window, stays at base fee", 1_000_000, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b20.NowFn = func() int64 { return e.LaunchTime + c.elapsed }
			assert.Equal(t, c.want, e.TotalFeeBps())
		})
	}
}

func TestQuoteIsSpecified(t *testing.T) {
	cases := []struct {
		name             string
		tokenIsCurrency0 bool
		zeroForOne       bool
		want             bool
	}{
		{"token=c1, sell c0(quote) for c1(token): quote is specified (buying token)", false, true, true},
		{"token=c1, sell c1(token) for c0(quote): token is specified (selling token)", false, false, false},
		{"token=c0, sell c0(token) for c1(quote): token is specified (selling token)", true, true, false},
		{"token=c0, sell c1(quote) for c0(token): quote is specified (buying token)", true, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := b20.Extra{TokenIsCurrency0: c.tokenIsCurrency0}
			assert.Equal(t, c.want, e.QuoteIsSpecified(c.zeroForOne))
		})
	}
}

// Buying the token with quote-as-input: LaunchHook.sol charges the fee pre-swap
// (beforeSwap), floor-rounded, reducing what the AMM sees as input.
func TestBeforeSwap_QuoteSpecified_ChargesFeeOnInput(t *testing.T) {
	e := refExtra()
	b20.NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds } // post-window: flat 100bps
	h := &Hook{Extra: e}

	// tokenIsCurrency0=false -> quote is currency0 -> zeroForOne=true sells quote in.
	amountIn := big.NewInt(1_000_000)
	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: amountIn})
	require.NoError(t, err)

	// floor(1_000_000 * 100 / 10_000) = 10_000, matching FeeAmount's floor mulDiv.
	assert.Equal(t, big.NewInt(10_000), result.DeltaSpecified)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
}

// Selling the token for quote: the fee must NOT be charged in BeforeSwap (it's
// deferred to AfterSwap on the realized output) -- charging it here too would
// double-count against AfterSwap's own fee application.
func TestBeforeSwap_TokenSpecified_NoFeeHere(t *testing.T) {
	e := refExtra()
	b20.NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
	h := &Hook{Extra: e}

	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: big.NewInt(1_000_000)})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
}

// Selling the token for quote: fee lands post-swap on the realized quote output.
func TestAfterSwap_TokenSpecified_ChargesFeeOnOutput(t *testing.T) {
	e := refExtra()
	b20.NowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
	h := &Hook{Extra: e}

	amountOut := big.NewInt(1_000_000)
	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        amountOut,
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(10_000), result.HookFee)
}

func TestBeforeSwap_ExactOutUnsupported(t *testing.T) {
	h := &Hook{Extra: refExtra()}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(1)})
	assert.ErrorIs(t, err, b20.ErrCalcInUnsupported)
}

// A pool whose Track() never succeeded (RPC failure, or a factory that constructs the
// Hook before HookExtra is populated) must refuse to quote rather than default to a
// zero-value Extra (BaseFeeBps=0), which would silently price the pool as fee-free.
func TestBeforeAfterSwap_UntrackedPool_Errors(t *testing.T) {
	h := &Hook{} // zero-value Extra: LaunchTime == 0

	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1_000_000)})
	assert.ErrorIs(t, err, b20.ErrNotTracked)

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.ErrorIs(t, err, b20.ErrNotTracked)
}
