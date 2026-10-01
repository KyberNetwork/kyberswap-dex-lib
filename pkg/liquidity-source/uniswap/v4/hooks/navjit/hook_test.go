package navjit

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// A ladder shaped like the live LOT-two venue on 2026-10-01 (block 77346769): USDG (6 dp) in,
// LOT (18 dp) out, flat to $1k then bending with constituent slippage.
var buyLadder = []Rung{
	{In: u("1000000"), Out: u("991667573448564743"), Gas: 1_130_316},
	{In: u("100000000"), Out: u("99166757344856474324"), Gas: 1_130_794},
	{In: u("1000000000"), Out: u("991568512243632021814"), Gas: 1_130_454},
	{In: u("20000000000"), Out: u("19793764531416765581247"), Gas: 1_193_710},
}

// LOT in, USDG out; float 49.27 LOT, ladder topped just under it.
var sellLadder = []Rung{
	{In: u("1000000000000000000"), Out: u("991370"), Gas: 887_439},
	{In: u("10000000000000000000"), Out: u("9913706"), Gas: 887_439},
	{In: u("47790000000000000000"), Out: u("47377000"), Gas: 885_300},
}

func trackedHook() *Hook {
	return &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NavJit},
		Extra: Extra{Tracked: true, UsdgIs0: true, Float: u("49265362021832501903"),
			Buy: buyLadder, Sell: sellLadder, Block: 77346769}}
}

func exactIn(zeroForOne bool, amt string) *uniswapv4.BeforeSwapParams {
	a, _ := new(big.Int).SetString(amt, 10)
	return &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: a}
}

func exactOut(zeroForOne bool, amt string) *uniswapv4.BeforeSwapParams {
	a, _ := new(big.Int).SetString(amt, 10)
	return &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: zeroForOne, AmountSpecified: a}
}

// The plugin takes the WHOLE specified amount, so the pool's own CL math (which says nothing about
// a NavJit fill) runs on zero, and pays the ladder output through the unspecified delta.
func TestBeforeSwap_ExactIn_TakesWholeAmountAtTheRung(t *testing.T) {
	r, err := trackedHook().BeforeSwap(exactIn(true, "100000000"))
	require.NoError(t, err)
	assert.Equal(t, "100000000", r.DeltaSpecified.String())
	assert.Equal(t, "-99166757344856474324", r.DeltaUnspecified.String(), "exactly the quoter's rung")
	assert.Equal(t, int64(1_130_794)-uniswapv4.DefaultGas.BaseGas, r.Gas)
	assert.Zero(t, r.SwapFee)
}

func TestBeforeSwap_ExactIn_InterpolatesBetweenRungs(t *testing.T) {
	// halfway between the $1k and $20k rungs (10.5k USDG)
	r, err := trackedHook().BeforeSwap(exactIn(true, "10500000000"))
	require.NoError(t, err)
	// lo.Out + (amt-lo.In)*(hi.Out-lo.Out)/(hi.In-lo.In), floored
	lo, hi := buyLadder[2], buyLadder[3]
	want := new(big.Int).Sub(hi.Out.ToBig(), lo.Out.ToBig())
	want.Mul(want, big.NewInt(9_500_000_000))
	want.Div(want, big.NewInt(19_000_000_000))
	want.Add(want, lo.Out.ToBig())
	assert.Equal(t, new(big.Int).Neg(want).String(), r.DeltaUnspecified.String())
	assert.Equal(t, hi.Gas-uniswapv4.DefaultGas.BaseGas, r.Gas, "gas of the upper rung")
}

func TestBeforeSwap_ExactIn_BelowFirstRungIsProportional(t *testing.T) {
	r, err := trackedHook().BeforeSwap(exactIn(true, "250000")) // $0.25
	require.NoError(t, err)
	assert.Equal(t, "-247916893362141185", r.DeltaUnspecified.String()) // floor(0.25 * rung0)
}

func TestBeforeSwap_Sell_IsTheOtherDirection(t *testing.T) {
	r, err := trackedHook().BeforeSwap(exactIn(false, "10000000000000000000"))
	require.NoError(t, err)
	assert.Equal(t, "-9913706", r.DeltaUnspecified.String())
}

// USDG as currency1 flips which zeroForOne is the buy.
func TestBeforeSwap_UsdgIsCurrency1(t *testing.T) {
	h := trackedHook()
	h.UsdgIs0 = false
	r, err := h.BeforeSwap(exactIn(false, "100000000")) // oneForZero = USDG in = buy
	require.NoError(t, err)
	assert.Equal(t, "-99166757344856474324", r.DeltaUnspecified.String())
}

func TestBeforeSwap_Refusals(t *testing.T) {
	h := trackedHook()
	_, err := h.BeforeSwap(exactIn(true, "20000000001"))
	assert.ErrorIs(t, err, ErrBeyondLadder, "one wei above the top buy rung")

	_, err = h.BeforeSwap(exactIn(false, "49265362021832501904"))
	assert.ErrorIs(t, err, ErrSellExceedsFloat, "one wei above the PoolManager float")

	_, err = h.BeforeSwap(exactIn(false, "48000000000000000000"))
	assert.ErrorIs(t, err, ErrBeyondLadder, "under the float but above the top quoted sell")

	_, err = h.BeforeSwap(exactIn(false, "1")) // 1 wei of LOT is worth 0 USDG units
	assert.ErrorIs(t, err, ErrZeroOutput)
	_, err = h.BeforeSwap(exactIn(true, "0"))
	assert.Error(t, err)

	h.Sell = nil
	_, err = h.BeforeSwap(exactIn(false, "1000000000000000000"))
	assert.ErrorIs(t, err, ErrEmptyLadder)

	_, err = (&Hook{}).BeforeSwap(exactIn(true, "1000000"))
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}

// CalcIn inverts the same chords, rounding the input UP, so feeding the result back through
// exact-in returns at least the requested output.
func TestBeforeSwap_ExactOut_InvertsTheLadder(t *testing.T) {
	h := trackedHook()
	for _, want := range []string{"1", "500000000000000000", "99166757344856474324", "5000000000000000000000",
		"19793764531416765581247"} {
		r, err := h.BeforeSwap(exactOut(true, want))
		require.NoError(t, err, want)
		assert.Equal(t, "-"+want, r.DeltaSpecified.String())
		back, err := h.BeforeSwap(exactIn(true, r.DeltaUnspecified.String()))
		require.NoError(t, err)
		got := new(big.Int).Neg(back.DeltaUnspecified)
		w, _ := new(big.Int).SetString(want, 10)
		assert.True(t, got.Cmp(w) >= 0, "round trip %s -> in %s -> %s", want, r.DeltaUnspecified, got)
	}
	_, err := h.BeforeSwap(exactOut(true, "19793764531416765581248"))
	assert.ErrorIs(t, err, ErrBeyondLadder)
}

func TestGetReserves_ReportsTheTopRungs(t *testing.T) {
	res, err := trackedHook().GetReserves(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, entity.PoolReserves{"47377000", "19793764531416765581247"}, res)

	h := trackedHook()
	h.UsdgIs0 = false
	res, _ = h.GetReserves(nil, nil)
	assert.Equal(t, entity.PoolReserves{"19793764531416765581247", "47377000"}, res)
}

func TestToLadder_StopsAtTheFirstRevertOrNonMonotoneRung(t *testing.T) {
	in := []*big.Int{big.NewInt(1), big.NewInt(10), big.NewInt(100), big.NewInt(1000)}
	out := []quoteResult{{big.NewInt(2), big.NewInt(5)}, {big.NewInt(20), big.NewInt(5)},
		{big.NewInt(200), big.NewInt(5)}, {big.NewInt(2000), big.NewInt(5)}}
	assert.Len(t, toLadder(in, out, []bool{true, true, false, true}), 2, "a revert ends the ladder")
	out[2].AmountOut = big.NewInt(20)
	assert.Len(t, toLadder(in, out, []bool{true, true, true, true}), 2, "a flat rung ends the ladder")
	assert.Len(t, toLadder(in, out, []bool{false, true, true, true}), 0)
}

// The extra round-trips through JSON, which is what the pool-service persists between Track and
// the simulator.
func TestExtra_JSONRoundTrip(t *testing.T) {
	raw, err := json.Marshal(trackedHook())
	require.NoError(t, err)
	var back Extra
	require.NoError(t, uniswapv4.HookExtra(raw).Unmarshal(&back))
	assert.Equal(t, trackedHook().Extra, back)
}

// End to end through uniswap-v4's PoolSimulator: an entity.Pool keyed at the NavJit hook gets this
// plugin, quotes the ladder in both directions and both swap types, and survives UpdateBalance /
// CloneState, with a standing position in the ticks that the plugin must ignore.
func TestPoolSimulator_UsesThePlugin(t *testing.T) {
	hookExtra, err := json.Marshal(trackedHook())
	require.NoError(t, err)
	staticExtra, _ := json.Marshal(uniswapv4.StaticExtra{Fee: 0, TickSpacing: 1, HooksAddress: HookAddresses[0]})
	sqrt, _ := new(big.Int).SetString("79438280045202668386950967478741068", 10) // tick 276377, ~1 LOT = $1.00
	extra, _ := json.Marshal(uniswapv4.Extra{Extra: &uniswapv3.Extra{
		Liquidity: big.NewInt(4_917_000_000_000_000), SqrtPriceX96: sqrt, TickSpacing: 1, Tick: big.NewInt(276377),
		Ticks: []uniswapv3.Tick{
			{Index: 276277, LiquidityGross: big.NewInt(4_917_000_000_000_000), LiquidityNet: big.NewInt(4_917_000_000_000_000)},
			{Index: 276477, LiquidityGross: big.NewInt(4_917_000_000_000_000), LiquidityNet: big.NewInt(-4_917_000_000_000_000)},
		}}, HookExtra: hookExtra})
	lot := "0xf5e660dc904b5017ff2cecc22608567c2be6d7c6"
	usdg := "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
	ep := entity.Pool{
		Address:     "0x8327ac000f8b77d690b8315290ba7dcb4834b12b377b0367234b3d6661f20056",
		Exchange:    valueobject.ExchangeUniswapV4NavJit,
		Type:        uniswapv4.DexType,
		Tokens:      []*entity.PoolToken{{Address: usdg, Decimals: 6, Swappable: true}, {Address: lot, Decimals: 18, Swappable: true}},
		Reserves:    entity.PoolReserves{"47377000", "19793764531416765581247"},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	}
	sim, err := uniswapv4.NewPoolSimulator(ep, valueobject.ChainIDRobinhood)
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdg, Amount: big.NewInt(100_000_000)}, TokenOut: lot})
	require.NoError(t, err)
	assert.Equal(t, "99166757344856474324", res.TokenAmountOut.Amount.String(),
		"exactly the ladder: the standing ticks contribute nothing")
	assert.Equal(t, int64(1_130_794), res.Gas, "the quoter's gas estimate for the whole swap")

	res, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: lot, Amount: big.NewInt(1e18)}, TokenOut: usdg})
	require.NoError(t, err)
	assert.Equal(t, "991370", res.TokenAmountOut.Amount.String())

	inRes, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: lot, Amount: big.NewInt(1e18)}, TokenIn: usdg})
	require.NoError(t, err)
	assert.Equal(t, "1008403", inRes.TokenAmountIn.Amount.String()) // ceil(1e18 * 1e6 / 991667573448564743)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdg, Amount: big.NewInt(30_000_000_000)}, TokenOut: lot})
	assert.ErrorIs(t, err, ErrBeyondLadder)

	// UpdateBalance + CloneState leave the quote unchanged (each swap fills at NAV, nothing depletes).
	cloned := sim.CloneState()
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: lot, Amount: big.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmount{Token: usdg, Amount: res.TokenAmountOut.Amount},
		SwapInfo:       res.SwapInfo,
	})
	for _, s := range []pool.IPoolSimulator{sim, cloned} {
		again, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: lot, Amount: big.NewInt(1e18)}, TokenOut: usdg})
		require.NoError(t, err)
		assert.Equal(t, "991370", again.TokenAmountOut.Amount.String())
	}
}

func TestRegistration(t *testing.T) {
	h, ok := uniswapv4.GetHook(common.HexToAddress("0x99a670d2103e1e4ebd53a483f725dd651a08eae0"), nil)
	require.True(t, ok)
	assert.Equal(t, valueobject.ExchangeUniswapV4NavJit, h.GetExchange())
	assert.True(t, h.AllowEmptyTicks())
	assert.True(t, uniswapv4.HasSwapPermissions(HookAddresses[0]))
	assert.True(t, h.CanBeforeSwap(HookAddresses[0]))
}
