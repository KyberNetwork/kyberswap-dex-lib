package arcade

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func graduatedPumpV2(usdcIsCurrency0 bool) Extra {
	e := graduatedPump(usdcIsCurrency0)
	e.NoSwapDelta = true
	return e
}

// The generation comes from the hook address: v1 carries both RETURNS_DELTA permission
// bits (0x3ECE), v2 neither (0x3EC2), and every registered address has an entry.
func TestGenerations(t *testing.T) {
	permissions := func(a common.Address) uint16 { return (uint16(a[18])<<8 | uint16(a[19])) & 0x3FFF }
	assert.Equal(t, uint16(0x3ECE), permissions(HookV1))
	assert.Equal(t, uint16(0x3EC2), permissions(HookV2))

	assert.Equal(t, []common.Address{HookV1, HookV2}, HookAddresses)
	assert.Len(t, Generations, len(HookAddresses))
	assert.False(t, Generations[HookV1].NoSwapDelta)
	assert.True(t, Generations[HookV2].NoSwapDelta)

	// The factory trusts the address over whatever extra was persisted.
	for _, c := range []struct {
		hook  common.Address
		extra string
		want  bool
	}{
		{HookV1, `{"tr":true,"m":0,"s":2}`, false},
		{HookV1, `{"tr":true,"m":0,"s":2,"nd":true}`, false},
		{HookV2, `{"tr":true,"m":0,"s":2}`, true},
		{HookV2, `{"tr":true,"m":0,"s":2,"nd":true}`, true},
		{HookV2, ``, true}, // not tracked yet
	} {
		h, ok := uniswapv4.GetHook(c.hook, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(c.extra)})
		require.True(t, ok)
		assert.Equal(t, c.want, h.(*Hook).NoSwapDelta, "%s %s", c.hook, c.extra)
		assert.Equal(t, "uniswap-v4-arcade", h.GetExchange())
	}
}

// v2: a graduated PUMP pool pays its fee as the pool's static LP fee. The hook takes
// nothing on either side, for both currency orderings, whatever an oracle would say.
func TestPumpV2_NoHookFee(t *testing.T) {
	for _, usdc0 := range []bool{true, false} {
		e := graduatedPumpV2(usdc0)
		e.ObsInit, e.EmaTickE3, e.GradMcapTick = true, 60_000_000, 50_000 // never read on v2
		h := &Hook{Extra: e}
		for _, zeroForOne := range []bool{true, false} { // a buy and a sell
			for _, calcOut := range []bool{true, false} { // exact-in and exact-out
				swap := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: zeroForOne, AmountSpecified: big.NewInt(1_234_567_891)}
				before, err := h.BeforeSwap(swap)
				require.NoError(t, err)
				assert.Equal(t, bignumber.ZeroBI, before.DeltaSpecified)
				assert.Equal(t, bignumber.ZeroBI, before.DeltaUnspecified)
				assert.Zero(t, before.SwapFee, "no fee override: the PoolKey fee stands")

				after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
					BeforeSwapParams: swap, AmountIn: big.NewInt(1_234_567_891), AmountOut: big.NewInt(987_654_321),
				})
				require.NoError(t, err)
				assert.Equal(t, bignumber.ZeroBI, after.HookFee)
			}
		}
	}
}

// v2 CLANKER / RWA behave as on v1: no hook fee, and the launch-window buy cap, which
// exact-out meets too (the token output is then the specified amount).
func TestDirectLaunchV2_BuyCap(t *testing.T) {
	orig := NowFn
	defer func() { NowFn = orig }()
	NowFn = func() int64 { return 1_000 + 30 } // first minute: cap = 1% = 10M tokens

	for _, mode := range []uint8{ModeClanker, ModeRwa} {
		h := &Hook{Extra: Extra{Tracked: true, NoSwapDelta: true, Mode: mode, Status: StatusGraduated, QuoteIsCurrency0: true, LaunchedAt: 1_000, BuyCapEnabled: true}}
		for _, calcOut := range []bool{true, false} {
			buy := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: true}
			after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: tokens(10_000_000)})
			require.NoError(t, err)
			assert.Equal(t, bignumber.ZeroBI, after.HookFee)

			_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: buy, AmountOut: new(big.Int).Add(tokens(10_000_000), big.NewInt(1))})
			assert.ErrorIs(t, err, ErrBuyExceedsCap)

			sell := &uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: false}
			_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: sell, AmountOut: tokens(900_000_000)})
			require.NoError(t, err, "sells are never capped")
		}
	}
}

// v2 keeps the curve guard (beforeSwap reverts while Curving or GraduationStarted), and
// exact-out stays refused on v1 only.
func TestV2_RefusalsAndExactOut(t *testing.T) {
	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1_000)}

	_, err := (&Hook{Extra: Extra{NoSwapDelta: true}}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrNotTracked)

	for _, status := range []uint8{StatusCurving, StatusGraduationStarted} {
		_, err = (&Hook{Extra: Extra{Tracked: true, NoSwapDelta: true, Mode: ModePump, Status: status}}).BeforeSwap(params)
		assert.ErrorIs(t, err, ErrNotGraduated)
	}

	_, err = (&Hook{Extra: Extra{Tracked: true, NoSwapDelta: true, Mode: 2, Status: StatusGraduated}}).BeforeSwap(params)
	assert.ErrorIs(t, err, ErrUnknownMode)

	exactOut := &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: true, AmountSpecified: big.NewInt(1)}
	_, err = (&Hook{Extra: graduatedPumpV2(true)}).BeforeSwap(exactOut)
	require.NoError(t, err)
	for _, v1 := range []Extra{graduatedPump(true), {Tracked: true, Mode: ModeClanker, Status: StatusGraduated}} {
		_, err = (&Hook{Extra: v1}).BeforeSwap(exactOut)
		assert.ErrorIs(t, err, ErrCalcInUnsupported)
	}
}
