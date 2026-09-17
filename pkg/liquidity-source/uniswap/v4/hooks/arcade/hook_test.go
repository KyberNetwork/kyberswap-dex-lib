package arcade

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func tokens(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }

func graduatedPump(usdcIsCurrency0 bool) Extra {
	return Extra{Tracked: true, Mode: ModePump, Status: StatusGraduated, UsdcIsCurrency0: usdcIsCurrency0}
}

func TestPumpFeeBps(t *testing.T) {
	cases := []struct {
		name string
		e    Extra
		want int64
	}{
		{"oracle not seeded charges the max", Extra{ObsInit: false}, 100},
		{"at the graduation mcap", Extra{ObsInit: true, EmaTickE3: 50_000_000, GradMcapTick: 50_000}, 100},
		{"below the graduation mcap", Extra{ObsInit: true, EmaTickE3: 49_000_000, GradMcapTick: 50_000}, 100},
		{"half way to the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 11_513) * 1_000, GradMcapTick: 50_000}, 100 - (70*11_513)/23_026},
		{"one tick short of the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 23_025) * 1_000, GradMcapTick: 50_000}, 100 - (70*23_025)/23_026},
		{"at the floor", Extra{ObsInit: true, EmaTickE3: (50_000 + 23_026) * 1_000, GradMcapTick: 50_000}, 30},
		{"far above the floor", Extra{ObsInit: true, EmaTickE3: 900_000_000, GradMcapTick: 50_000}, 30},
		// int256(emaTickE3) / 1000 truncates toward zero in Solidity, as in Go:
		// -39_999_500 / 1000 = -39_999, growth = -39_999 - (-50_000) = 10_001.
		{"negative ticks truncate toward zero", Extra{ObsInit: true, EmaTickE3: -39_999_500, GradMcapTick: -50_000}, 100 - (70*10_001)/23_026},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.e.PumpFeeBps())
		})
	}
}

// Buying with USDC exact-in: the fee comes off the specified USDC input (beforeSwap).
func TestPump_BuyTakesFeeOnInput(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		h := &Hook{Extra: graduatedPump(usdc0)}
		zeroForOne := usdc0 // USDC in
		in := big.NewInt(1_234_567_891)

		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: in})
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(12_345_678), before.DeltaSpecified) // floor(1_234_567_891 * 100 / 10_000)
		assert.Equal(t, bignumber.ZeroBI, before.DeltaUnspecified)

		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne},
			AmountOut:        tokens(1_000),
		})
		require.NoError(t, err)
		assert.Equal(t, bignumber.ZeroBI, after.HookFee)
	}
}

// Selling the token exact-in: nothing before the swap, the fee comes off the USDC output.
func TestPump_SellTakesFeeOnOutput(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		e := graduatedPump(usdc0)
		e.ObsInit, e.EmaTickE3, e.GradMcapTick = true, 60_000_000, 50_000 // growth 10_000 ticks
		h := &Hook{Extra: e}
		zeroForOne := !usdc0 // token in

		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: tokens(5)})
		require.NoError(t, err)
		assert.Equal(t, bignumber.ZeroBI, before.DeltaSpecified)

		out := big.NewInt(987_654_321)
		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne},
			AmountOut:        out,
		})
		require.NoError(t, err)
		feeBps := int64(100 - (70*10_000)/23_026) // 70
		want := new(big.Int).Div(new(big.Int).Mul(out, big.NewInt(feeBps)), big.NewInt(10_000))
		assert.Equal(t, want, after.HookFee)
	}
}

// CLANKER / RWA: the fee is the pool's native LP fee, the hook takes nothing.
func TestDirectLaunch_NoHookFee(t *testing.T) {
	NowFn = func() int64 { return 2_000_000_000 }
	for _, mode := range []uint8{ModeClanker, ModeRwa} {
		h := &Hook{Extra: Extra{Tracked: true, Mode: mode, Status: StatusGraduated, QuoteIsCurrency0: true, LaunchedAt: 1_000, BuyCapEnabled: true}}
		before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1e9)})
		require.NoError(t, err)
		assert.Equal(t, bignumber.ZeroBI, before.DeltaSpecified)
		after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
			BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true},
			AmountOut:        tokens(500_000_000), // far above any cap, but long after the window
		})
		require.NoError(t, err)
		assert.Equal(t, bignumber.ZeroBI, after.HookFee)
	}
}

func TestPerTxMaxBuyTokens(t *testing.T) {
	e := Extra{LaunchedAt: 1_000}
	cases := []struct {
		elapsed int64
		want    *big.Int
	}{
		{0, tokens(10_000_000)},   // 1%
		{59, tokens(10_000_000)},  // still the first minute
		{60, tokens(20_000_000)},  // 2%
		{240, tokens(50_000_000)}, // 5%
		{299, tokens(50_000_000)}, // last second of the window
		{300, nil},                // uncapped
		{10_000, nil},
	}
	for _, c := range cases {
		got := e.PerTxMaxBuyTokens(1_000 + c.elapsed)
		if c.want == nil {
			assert.Nil(t, got, "elapsed %d", c.elapsed)
		} else {
			assert.Equal(t, c.want, got, "elapsed %d", c.elapsed)
		}
	}
}

func TestDirectLaunch_BuyCap(t *testing.T) {
	h := &Hook{Extra: Extra{Tracked: true, Mode: ModeClanker, Status: StatusGraduated, QuoteIsCurrency0: false, LaunchedAt: 1_000, BuyCapEnabled: true}}
	// cap = 1% = 10M tokens; the quote is currency1, so a buy is oneForZero.
	NowFn = func() int64 { return 1_000 + 30 }
	buy := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false}

	_, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: tokens(10_000_000)})
	require.NoError(t, err, "exactly at the cap passes (the contract reverts only above it)")

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: new(big.Int).Add(tokens(10_000_000), big.NewInt(1))})
	assert.ErrorIs(t, err, ErrBuyExceedsCap)

	sell := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true}
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: sell, AmountOut: tokens(900_000_000)})
	require.NoError(t, err, "sells are never capped")

	h.BuyCapEnabled = false
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: tokens(900_000_000)})
	require.NoError(t, err, "owner-disabled cap")
}

func TestRefusesUntradablePools(t *testing.T) {
	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1_000)}

	_, err := (&Hook{}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrNotTracked)

	for _, status := range []uint8{StatusCurving, StatusGraduationStarted} {
		_, err = (&Hook{Extra: Extra{Tracked: true, Mode: ModePump, Status: status}}).BeforeSwap(params)
		assert.ErrorIs(t, err, ErrNotGraduated)
	}

	_, err = (&Hook{Extra: Extra{Tracked: true, Mode: 2, Status: StatusGraduated}}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrUnknownMode)

	_, err = (&Hook{Extra: graduatedPump(true)}).BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(1)})
	assert.ErrorIs(t, err, ErrCalcInUnsupported)
}
