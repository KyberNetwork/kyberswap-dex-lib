package b20

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// reference values pulled from LaunchHook.sol's live poolConfig for a real B20 launch
// on Base (poolId 0x68d39022eee9e18f82fe929f70dd8d2009e442e6d80c6c1ffc170c32b7d3b671),
// so the decay math is checked against the actual on-chain fee schedule, not just its
// own logic restated.
func refExtra() Extra {
	return Extra{
		TokenIsCurrency0:       false,
		BaseFeeBps:             100,
		AntiSnipeStartTotalBps: 9900,
		AntiSnipeWindowSeconds: 16,
		LaunchTime:             1786986263,
	}
}

// totalFeeBps must reproduce LaunchHook.sol's _totalFeeBps exactly: a wrong bps here
// silently mis-prices every swap during the anti-snipe window (first 16s after
// launch for this reference pool) without erroring, so a route built on this quote
// would settle for less than it should on-chain.
func TestTotalFeeBps_Decay(t *testing.T) {
	h := &Hook{Extra: refExtra()}

	cases := []struct {
		name    string
		elapsed int64
		want    int64
	}{
		{"at launch instant, surcharge is maximal", 0, 9900},
		{"halfway through the window, surcharge is halved", 8, 5000},
		{"window boundary, decays to just base fee", 16, 100},
		{"long after the window, stays at base fee", 1_000_000, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nowFn = func() int64 { return h.LaunchTime + c.elapsed }
			assert.Equal(t, c.want, h.totalFeeBps())
		})
	}
}

// A misconfigured pool (antiSnipeStartTotalBps above the hard ceiling somehow making
// it through Track) must never let the simulator quote a fee above what the contract
// can ever charge -- MaxTotalFeeBps is the contract's own hard ceiling.
func TestTotalFeeBps_CappedAtMax(t *testing.T) {
	// On-chain _validateFeeConfig rejects antiSnipeStartTotalBps > MAX_TOTAL_FEE_BPS at
	// registration time, but the Go clamp must hold defensively too -- use an
	// out-of-policy value (10_000) so the surcharge math alone would exceed the cap and
	// the assertion can actually fail if the `if total > MaxTotalFeeBps` clamp is removed.
	h := &Hook{Extra: Extra{BaseFeeBps: 0, AntiSnipeStartTotalBps: 10_000, AntiSnipeWindowSeconds: 10, LaunchTime: 1}}
	nowFn = func() int64 { return 1 } // elapsed=0 -> surcharge=10_000, would exceed the cap uncapped
	assert.Equal(t, int64(MaxTotalFeeBps), h.totalFeeBps())
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
			h := &Hook{Extra: Extra{TokenIsCurrency0: c.tokenIsCurrency0}}
			assert.Equal(t, c.want, h.quoteIsSpecified(c.zeroForOne))
		})
	}
}

// Buying the token with quote-as-input: LaunchHook.sol charges the fee pre-swap
// (beforeSwap), floor-rounded, reducing what the AMM sees as input.
func TestBeforeSwap_QuoteSpecified_ChargesFeeOnInput(t *testing.T) {
	e := refExtra()
	nowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds } // post-window: flat 100bps
	h := &Hook{Extra: e}

	// tokenIsCurrency0=false -> quote is currency0 -> zeroForOne=true sells quote in.
	amountIn := big.NewInt(1_000_000)
	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: amountIn})
	require.NoError(t, err)

	// floor(1_000_000 * 100 / 10_000) = 10_000, matching _feeAmount's floor mulDiv.
	assert.Equal(t, big.NewInt(10_000), result.DeltaSpecified)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
}

// Selling the token for quote: the fee must NOT be charged in BeforeSwap (it's
// deferred to AfterSwap on the realized output) -- charging it here too would
// double-count against AfterSwap's own fee application.
func TestBeforeSwap_TokenSpecified_NoFeeHere(t *testing.T) {
	e := refExtra()
	nowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
	h := &Hook{Extra: e}

	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false, AmountSpecified: big.NewInt(1_000_000)})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
}

// Selling the token for quote: fee lands post-swap on the realized quote output.
func TestAfterSwap_TokenSpecified_ChargesFeeOnOutput(t *testing.T) {
	e := refExtra()
	nowFn = func() int64 { return e.LaunchTime + e.AntiSnipeWindowSeconds }
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
	assert.ErrorIs(t, err, errCalcInUnsupported)
}

// A pool whose Track() never succeeded (RPC failure, or a factory that constructs the
// Hook before HookExtra is populated) must refuse to quote rather than default to a
// zero-value Extra (BaseFeeBps=0), which would silently price the pool as fee-free.
func TestBeforeAfterSwap_UntrackedPool_Errors(t *testing.T) {
	h := &Hook{} // zero-value Extra: LaunchTime == 0

	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1_000_000)})
	assert.ErrorIs(t, err, errNotTracked)

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.ErrorIs(t, err, errNotTracked)
}
