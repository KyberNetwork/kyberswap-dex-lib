package premium

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/hooktest"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestHookRegistration(t *testing.T) {
	t.Parallel()

	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, ok, "hook should be registered for %s", addr.Hex())
		assert.Equal(t, valueobject.ExchangeUniswapV4Prm, hook.GetExchange())
	}
}

func TestHookFactory_WithExtra(t *testing.T) {
	t.Parallel()

	extra := Hook{MemeIsCurrency0: true, Paused: true}
	extraBytes, _ := json.Marshal(extra)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})
	require.True(t, ok)

	premiumHook, ok := hook.(*Hook)
	require.True(t, ok)
	assert.True(t, premiumHook.MemeIsCurrency0)
	assert.True(t, premiumHook.Paused)
}

// TestBeforeSwap_DeskSpecified exercises PremiumLaunchHook._beforeSwap's "desk is the
// specified (input) currency" branch: fee is taken up front, floor(specified * 1%),
// reducing what actually reaches the underlying curve. Meme is currency1 here
// (MemeIsCurrency0=false), so zeroForOne (spending currency0=desk) means desk is specified.
func TestBeforeSwap_DeskSpecified(t *testing.T) {
	t.Parallel()

	extra := Hook{MemeIsCurrency0: false}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	res, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true, // spending currency0 = desk -> desk is specified
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	// fee = floor(1_000_000 * 100 / 10_000) = 10_000
	assert.Equal(t, int64(10_000), res.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), res.DeltaUnspecified.Int64())
}

// TestBeforeSwap_MemeSpecified exercises the opposite branch: meme is the specified
// (input) currency (a sell), so beforeSwap must be a no-op and AfterSwap takes the fee
// instead.
func TestBeforeSwap_MemeSpecified(t *testing.T) {
	t.Parallel()

	extra := Hook{MemeIsCurrency0: false}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	res, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      false, // spending currency1 = meme -> meme is specified
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), res.DeltaUnspecified.Int64())
}

// CalcAmountIn reverses exact-in (exact-output being disabled on-chain doesn't matter), so it must invert the
// desk fee whether desk is the input or the output.
// Goldens are the legacy CalcAmountIn output over the same amountOut grid the randomized sweep covered, for both
// MemeIsCurrency0 directions (28 cases each, 56 total).
func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		memeIsCurrency0 bool
		golden          map[int]map[int]map[string]string
	}{
		{false, goldenMemeIsCurrency0False},
		{true, goldenMemeIsCurrency0True},
	} {
		pSim := hooktest.NewPoolSimulator(t, HookAddresses[0], Hook{MemeIsCurrency0: tc.memeIsCurrency0})
		hooktest.RequireRoundTrip(t, pSim)
		testutil.TestCalcAmountInIPoolSim(t, pSim, tc.golden)
	}
}

var goldenMemeIsCurrency0False = map[int]map[int]map[string]string{
	0: {
		1: {
			"1":                            "19832383977",
			"1000":                         "19832360373413",
			"1000000":                      "19832360311867762",
			"1000000000":                   "19832360316442903314",
			"1000000000000":                "19832364961201583274220",
			"1000000000000000":             "19837010809060867668395449",
			"3000000000000000":             "59538955048938757105789147",
			"100000000000000000":           "2030846253522955297668325582",
			"500000000000000000":           "11232871424778186588600562265",
			"1000000000000000000":          "25905534196180652198603002502",
			"2000000000000000000":          "74679980674553743558580449052",
			"10000000000000000000":         "",
			"100000000000000000000000000":  "",
			"1000000000000000000000000000": "",
		},
	},
	1: {
		0: {
			"1":                            "2",
			"1000":                         "2",
			"1000000":                      "2",
			"1000000000":                   "2",
			"1000000000000":                "53",
			"1000000000000000":             "51758",
			"3000000000000000":             "155270",
			"100000000000000000":           "5175650",
			"500000000000000000":           "25878250",
			"1000000000000000000":          "51756500",
			"2000000000000000000":          "103512998",
			"10000000000000000000":         "517564989",
			"100000000000000000000000000":  "5181918531977827",
			"1000000000000000000000000000": "52390272045794225",
		},
	},
}

var goldenMemeIsCurrency0True = map[int]map[int]map[string]string{
	0: {
		1: {
			"1":                            "19634060138",
			"1000":                         "19830377223156",
			"1000000":                      "19832360113561897",
			"1000000000":                   "19832360314506616672",
			"1000000000000":                "19832365008165114443006",
			"1000000000000000":             "19837057794907147474980788",
			"3000000000000000":             "59539378320479219110523361",
			"100000000000000000":           "2031338829220706078761003737",
			"500000000000000000":           "11247957560892654030441293811",
			"1000000000000000000":          "25985913474991172163579281354",
			"2000000000000000000":          "75351891940375807381776464367",
			"10000000000000000000":         "",
			"100000000000000000000000000":  "",
			"1000000000000000000000000000": "",
		},
	},
	1: {
		0: {
			"1":                            "2",
			"1000":                         "2",
			"1000000":                      "2",
			"1000000000":                   "2",
			"1000000000000":                "53",
			"1000000000000000":             "51757",
			"3000000000000000":             "155270",
			"100000000000000000":           "5175650",
			"500000000000000000":           "25878250",
			"1000000000000000000":          "51756499",
			"2000000000000000000":          "103512998",
			"10000000000000000000":         "517564989",
			"100000000000000000000000000":  "5181855770346473",
			"1000000000000000000000000000": "52383857491998156",
		},
	},
}

func TestBeforeSwap_Paused(t *testing.T) {
	t.Parallel()

	extra := Hook{Paused: true}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	_, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.ErrorIs(t, err, ErrPoolPaused)
}

// TestAfterSwap_DeskSpecified_Noop mirrors PremiumLaunchHook._afterSwap's deskIsSpecified
// branch returning int128(0) - the fee was already taken in BeforeSwap, not again here.
func TestAfterSwap_DeskSpecified_Noop(t *testing.T) {
	t.Parallel()

	extra := Hook{MemeIsCurrency0: false}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	res, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true, // desk specified
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(990_000),
		AmountOut: big.NewInt(500_000),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.HookFee.Int64())
}

// TestAfterSwap_MemeSpecified_TakesFee mirrors the non-deskIsSpecified branch: fee is
// floor(realized desk output * 1%), reducing what the trader actually receives.
func TestAfterSwap_MemeSpecified_TakesFee(t *testing.T) {
	t.Parallel()

	extra := Hook{MemeIsCurrency0: false}
	extraBytes, _ := json.Marshal(extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extraBytes})

	res, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      false, // spending currency1 = meme (MemeIsCurrency0=false) -> meme is specified
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(1_000_000),
		AmountOut: big.NewInt(1_000_000), // realized desk output this leg
	})
	require.NoError(t, err)
	// fee = floor(1_000_000 * 100 / 10_000) = 10_000
	assert.Equal(t, int64(10_000), res.HookFee.Int64())
}

func TestCloneState_DeepCopy(t *testing.T) {
	t.Parallel()

	original := &Hook{
		Hook:            &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Prm},
		MemeIsCurrency0: true,
		Paused:          false,
	}
	cloned := original.CloneState().(*Hook)
	cloned.Paused = true

	assert.False(t, original.Paused)
	assert.True(t, cloned.Paused)
}
