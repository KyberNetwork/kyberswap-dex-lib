package premium

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
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

func TestBeforeSwap_ExactOutputDisabled(t *testing.T) {
	t.Parallel()

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{})
	_, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         false, // exact-output
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.ErrorIs(t, err, ErrExactOutputDisabled)
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
