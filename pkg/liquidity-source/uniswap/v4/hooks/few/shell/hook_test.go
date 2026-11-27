package fewshell

import (
	"context"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func calcOut(t testing.TB, sim *uniswapv4.PoolSimulator, tokenIn, tokenOut string, amount string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big10(amount)}, TokenOut: tokenOut})
}

// The four exact-input outputs the chain returned at block 26504066 for 1e12..1e15 wei WETH: the pair's 997/1000 curve
// less a 10 bps fee rounded up.
func TestCalcAmountOut_OnChainNumbers(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{0: {1: {
		"1000000000000": "1721", "10000000000000": "17214", "100000000000000": "172147", "1000000000000000": "1721480",
	}}})
}

// With the protocol fee off the output is the pair's, with no fee rounding at all.
func TestCalcAmountOut_ProtocolFeeOff(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), 0)
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{0: {1: {
		"1000000000000": "1723", "10000000000000": "17232", "100000000000000": "172320", "1000000000000000": "1723204",
	}}})
}

// Each direction pays its own protocol fee: pF packs the oneForZero fee above the zeroForOne one.
func TestCalcAmountOut_FeeIsPerDirection(t *testing.T) {
	t.Parallel()
	pF := uint64(3000<<24 | 20<<12 | 40) // zeroForOne 40 (1000 ppm), oneForZero 20 (500 ppm)
	sim := newSim(t, liveExtra(), pF)

	res, err := calcOut(t, sim, weth, usdt, "1000000000000000") // fwOut 1723204: fee ceil(1723.204)=1724
	require.NoError(t, err)
	assert.Equal(t, "1721480", res.TokenAmountOut.Amount.String())

	// oneForZero: 1e6 USDT in. fwOut at 500 ppm: fee = ceil(fwOut*500/1e6).
	rIn, rOut := liveExtra().Reserves[1], liveExtra().Reserves[0]
	fwOut, err := getAmountOut(n(1_000_000), &rIn, &rOut)
	require.NoError(t, err)
	want := new(uint256.Int).Sub(fwOut, protocolFee(fwOut, 500))
	res, err = calcOut(t, sim, usdt, weth, "1000000")
	require.NoError(t, err)
	assert.Equal(t, want.Dec(), res.TokenAmountOut.Amount.String())
	assert.NotEqual(t, new(uint256.Int).Sub(fwOut, protocolFee(fwOut, 1000)).Dec(), res.TokenAmountOut.Amount.String(),
		"oneForZero must not pay the zeroForOne fee")
}

// Reversed token order: the same pool traded from the other currency uses the other reserve as reserveIn.
func TestCalcAmountOut_BothDirections(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	ex := liveExtra()
	fwOut, err := getAmountOut(n(1_000_000), &ex.Reserves[1], &ex.Reserves[0])
	require.NoError(t, err)
	res, err := calcOut(t, sim, usdt, weth, "1000000")
	require.NoError(t, err)
	assert.Equal(t, new(uint256.Int).Sub(fwOut, protocolFee(fwOut, 1000)).Dec(), res.TokenAmountOut.Amount.String())
}

// Every revert condition of the contract is an error here, and only then.
func TestCalcAmountOut_Reverts(t *testing.T) {
	t.Parallel()
	tooBig := new(big.Int).Lsh(big.NewInt(1), 127).String()

	for name, tc := range map[string]struct {
		mutate func(*Extra)
		amount string
		want   error
	}{
		"paused":                 {func(e *Extra) { e.Paused = true }, "1000000000000", ErrPaused},
		"inventory of the input": {func(e *Extra) { e.Inventory[0] = *n(999_999_999_999) }, "1000000000000", ErrInsufficientInventory},
		"backing of the output":  {func(e *Extra) { e.Backing[1] = *n(1720) }, "1000000000000", ErrInsufficientBacking}, // fwOut is 1723
		"zero input reserve":     {func(e *Extra) { e.Reserves[0].Clear() }, "1000000000000", ErrDegeneratePair},
		"zero output reserve":    {func(e *Extra) { e.Reserves[1].Clear() }, "1000000000000", ErrDegeneratePair},
		"pair gives nothing":     {func(*Extra) {}, "1", ErrInsufficientAmount},
		"amount past int128":     {func(*Extra) {}, tooBig, ErrInvalidAmount},
		"zero amount":            {func(*Extra) {}, "0", ErrInvalidAmount},
		"not tracked (no pair)":  {func(e *Extra) { e.Pair = [20]byte{} }, "1000000000000", ErrPoolIsNotTracked},
		"reserve past uint112": {func(e *Extra) {
			e.Reserves = [2]uint256.Int{*uint256.MustFromDecimal("5192296858534827628530496329220090"), *uint256.MustFromDecimal("5192296858534827628530496329220095")}
		}, "10", ErrReserveOverflow}, // the pair's reserve is a uint112 (max 2^112-1)
		"net output zero (fee 1/1)": {func(e *Extra) { e.Reserves = [2]uint256.Int{*n(1_000_000_000_000_000), *n(3)} }, "1000000000000000", ErrInsufficientAmount},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := liveExtra()
			tc.mutate(&ex)
			sim := newSim(t, ex, livePF)
			_, err := calcOut(t, sim, weth, usdt, tc.amount)
			assert.ErrorIs(t, err, tc.want)
		})
	}

	// Past the input inventory by one wei, and exactly at it.
	ex := liveExtra()
	sim := newSim(t, ex, livePF)
	_, err := calcOut(t, sim, weth, usdt, ex.Inventory[0].Dec())
	assert.NoError(t, err, "an input equal to the inventory is takeable")
	_, err = calcOut(t, sim, weth, usdt, new(uint256.Int).AddUint64(&ex.Inventory[0], 1).Dec())
	assert.ErrorIs(t, err, ErrInsufficientInventory)

	// Backing equal to fwOut is enough, one less is not.
	ex = liveExtra()
	ex.Backing[1] = *n(1723)
	_, err = calcOut(t, newSim(t, ex, livePF), weth, usdt, "1000000000000")
	assert.NoError(t, err)
	ex.Backing[1] = *n(1722)
	_, err = calcOut(t, newSim(t, ex, livePF), weth, usdt, "1000000000000")
	assert.ErrorIs(t, err, ErrInsufficientBacking)
}

// A pool whose hX is another hook's format (the live pool's calibrator state before this hook shipped) has no pair
// yet: it must refuse to quote until Track has run, not price from a half-read struct.
func TestCalcAmountOut_CalibratorHXIsUntracked(t *testing.T) {
	t.Parallel()
	_, err := calcOut(t, newSim(t, calibratorHX, livePF), weth, usdt, "1000000000000")
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
	_, err = calcOut(t, newSim(t, nil, livePF), weth, usdt, "1000000000000")
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}

// CalcAmountIn reverses exact input: the least input whose CalcAmountOut gives at least the output, both directions.
func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	testutil.TestCalcAmountIn(t, sim)
}

// The inverse must apply the exact-in fee rule (on the output), not the contract's exact-out one (on the input); the two
// differ by about the fee squared, which is a whole unit at these sizes.
func TestCalcAmountIn_IsMinimalAgainstCalcAmountOut(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	for _, amount := range []string{"1721", "17214", "172147", "1721480", "1721481", "99999999"} {
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: big10(amount)}, TokenIn: weth})
		require.NoError(t, err, amount)
		in := res.TokenAmountIn.Amount
		out, err := calcOut(t, sim, weth, usdt, in.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(big10(amount)), 0, amount)
		less, err := calcOut(t, sim, weth, usdt, new(big.Int).Sub(in, big.NewInt(1)).String())
		assert.True(t, err != nil || less.TokenAmountOut.Amount.Cmp(big10(amount)) < 0, "one wei less must fall short: %s", amount)
	}
}

func TestCalcAmountIn_Errors(t *testing.T) {
	t.Parallel()
	ex := liveExtra()
	sim := newSim(t, ex, livePF)
	// The whole output reserve, and more, cannot be bought.
	for _, out := range []string{ex.Reserves[1].Dec(), "29261029832820"} {
		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: big10(out)}, TokenIn: weth})
		assert.Error(t, err, out)
	}
	// An output the input inventory cannot cover on the way in.
	ex.Inventory[0] = *n(1_000_000)
	_, err := newSim(t, ex, livePF).CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: big10("17214")}, TokenIn: weth})
	assert.ErrorIs(t, err, ErrInsufficientInventory)
	ex = liveExtra()
	ex.Paused = true
	_, err = newSim(t, ex, livePF).CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: big10("17214")}, TokenIn: weth})
	assert.ErrorIs(t, err, ErrPaused)
}

// A route that uses the pool twice must see the first swap: the pair's reserves and the wrappers' backing move, and the
// pool keeps quoting (its decoy reserves are replaced by the hook's, so the v3 balance check does not trip).
func TestUpdateBalance_SecondSwapSeesTheFirst(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	amountIn := "100000000000000000000" // 100 WETH
	first, err := calcOut(t, sim, weth, usdt, amountIn)
	require.NoError(t, err)

	cloned := sim.CloneState().(*uniswapv4.PoolSimulator)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: big10(amountIn)},
		TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})

	// The second quote is the closed form from the post-swap reserves.
	ex := liveExtra()
	fwOut, err := getAmountOut(uint256.MustFromDecimal(amountIn), &ex.Reserves[0], &ex.Reserves[1])
	require.NoError(t, err)
	post := Extra{Reserves: ex.Reserves}
	post.Reserves[0].Add(&post.Reserves[0], uint256.MustFromDecimal(amountIn))
	post.Reserves[1].Sub(&post.Reserves[1], fwOut)
	second, err := calcOut(t, sim, weth, usdt, amountIn)
	require.NoError(t, err)
	wantFw, err := getAmountOut(uint256.MustFromDecimal(amountIn), &post.Reserves[0], &post.Reserves[1])
	require.NoError(t, err)
	assert.Equal(t, new(uint256.Int).Sub(wantFw, protocolFee(wantFw, 1000)).Dec(), second.TokenAmountOut.Amount.String())
	assert.Equal(t, 1, first.TokenAmountOut.Amount.Cmp(second.TokenAmountOut.Amount), "price impact: the second swap gets less")

	// The clone taken before kept the old state; the quote has no side effects.
	again, err := calcOut(t, cloned, weth, usdt, amountIn)
	require.NoError(t, err)
	assert.Equal(t, first.TokenAmountOut.Amount, again.TokenAmountOut.Amount)

	// The backing moved too: the output wrapper lost fwOut, so a swap past what is left reverts.
	ex = liveExtra()
	ex.Backing[1] = *new(uint256.Int).AddUint64(fwOut, 10)
	small := newSim(t, ex, livePF)
	first, err = calcOut(t, small, weth, usdt, amountIn)
	require.NoError(t, err)
	small.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: big10(amountIn)},
		TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})
	_, err = calcOut(t, small, weth, usdt, amountIn)
	assert.ErrorIs(t, err, ErrInsufficientBacking)
}

// Swapping both ways many times over keeps the pool quotable: the pool's reserves are the hook's pseudo TVL, not the
// decoy position's ~$1.
func TestUpdateBalance_RepeatedSwapsStayQuotable(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	pseudo, err := (&Hook{Extra: liveExtra()}).GetReserves(context.Background(), nil)
	require.NoError(t, err)
	sim.Info.Reserves = []*big.Int{big10(pseudo[0]), big10(pseudo[1])}
	for range 5 {
		for _, dir := range [][3]string{{weth, usdt, "50000000000000000000"}, {usdt, weth, "20000000000"}} {
			res, err := calcOut(t, sim, dir[0], dir[1], dir[2])
			require.NoError(t, err)
			sim.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: dir[0], Amount: big10(dir[2])},
				TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
		}
	}
}

// GetReserves is pseudoTotalValueLocked: the pair's reserves capped by backing and PoolManager inventory.
func TestGetReserves_IsPseudoTVL(t *testing.T) {
	t.Parallel()
	ex := liveExtra()
	ex.Backing[0] = *n(1_000)
	ex.Inventory[1] = *n(5_000)
	got, err := (&Hook{Extra: ex}).GetReserves(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"1000", "5000"}, []string(got))

	ex = liveExtra() // the inventory of currency0 (564 WETH) is the tightest cap there; the pair's reserve of currency1
	assert.Equal(t, []string{ex.Inventory[0].Dec(), ex.Reserves[1].Dec()}, mustReserves(t, &Hook{Extra: ex}))

	ex.Reserves[1].Clear() // an empty side empties both, as in the contract
	assert.Equal(t, []string{"0", "0"}, mustReserves(t, &Hook{Extra: ex}))

	_, err = (&Hook{}).GetReserves(context.Background(), nil)
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}

func mustReserves(t testing.TB, h *Hook) []string {
	t.Helper()
	got, err := h.GetReserves(context.Background(), nil)
	require.NoError(t, err)
	return got
}

// The router's price limit must pass the hook's own checks whatever the decoy ticks are: MIN+1 and MAX-1.
func TestPriceLimitIsTheWidestOne(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	for _, tc := range []struct {
		tokenIn, tokenOut string
		want              string
	}{
		{weth, usdt, "4295128740"},
		{usdt, weth, "1461446703485210103287273052203988822378723970341"},
	} {
		meta := sim.GetMetaInfo(tc.tokenIn, tc.tokenOut).(uniswapv4.PoolMetaInfo)
		assert.Equal(t, tc.want, meta.PriceLimit.Dec())
	}
}

// The hook resolves to this package, not the fallback calibrator, and keeps the few wrapper hook's exchange. A pool the
// calibrator stored as uniswap-v4-fee moves to the official dex's uniswap-v4 at its next track (ExchangeOf with the
// tracker's dexID), as the few wrapper pools already are.
func TestRegisteredAndExchange(t *testing.T) {
	t.Parallel()
	h, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{})
	require.True(t, ok)
	require.IsType(t, &Hook{}, h)
	assert.True(t, h.CanBeforeSwap(HookAddresses[0]))
	assert.False(t, h.CanAfterSwap(HookAddresses[0]))
	assert.True(t, h.AllowEmptyTicks())
	assert.Equal(t, "uniswap-v4", h.GetExchange())
	assert.Equal(t, "uniswap-v4", uniswapv4.ExchangeOf(h, uniswapv4.DexType, "uniswap-v4-fee"))

	pl := newPool(t, liveExtra(), livePF)
	pl.Exchange = "uniswap-v4"
	sim, err := uniswapv4.NewPoolSimulator(pl, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	assert.Equal(t, "uniswap-v4", sim.GetExchange())
}

func TestPoolProtocolFee(t *testing.T) {
	t.Parallel()
	p := newPool(t, nil, livePF)
	assert.EqualValues(t, 40, uniswapv4.PoolProtocolFee(&p, true))
	assert.EqualValues(t, 40, uniswapv4.PoolProtocolFee(&p, false))
	p = newPool(t, nil, 3000<<24|20<<12|40)
	assert.EqualValues(t, 40, uniswapv4.PoolProtocolFee(&p, true))
	assert.EqualValues(t, 20, uniswapv4.PoolProtocolFee(&p, false))
	p = newPool(t, nil, 0)
	assert.EqualValues(t, 0, uniswapv4.PoolProtocolFee(&p, true))
	assert.EqualValues(t, 0, uniswapv4.PoolProtocolFee(nil, true))
}

// The decoy position's ticks must not narrow the router's limit: the hook rejects a limit on the wrong side of the FewV2
// pair's price, which the decoy ticks know nothing about.
func TestPriceLimitIgnoresDecoyTicks(t *testing.T) {
	t.Parallel()
	pl := newPoolWithTicks(t, liveExtra(), livePF, -198600, -198400)
	sim, err := uniswapv4.NewPoolSimulator(pl, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	require.NotEqual(t, "4295128740", sim.PoolSimulator.GetSqrtPriceLimit(true).Dec(), "the default limit follows the decoy ticks")
	assert.Equal(t, "4295128740", sim.GetMetaInfo(weth, usdt).(uniswapv4.PoolMetaInfo).PriceLimit.Dec())
	assert.Equal(t, "1461446703485210103287273052203988822378723970341",
		sim.GetMetaInfo(usdt, weth).(uniswapv4.PoolMetaInfo).PriceLimit.Dec())
}

// Amounts that overflow the contract's int128 (or a uint256 product) are errors for both quote directions.
func TestHugeAmountsAreErrors(t *testing.T) {
	t.Parallel()
	sim := newSim(t, liveExtra(), livePF)
	for _, bits := range []uint{127, 128, 200, 255} {
		amount := new(big.Int).Lsh(big.NewInt(1), bits)
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amount}, TokenOut: usdt})
		assert.ErrorIs(t, err, ErrInvalidAmount, "out 2^%d", bits)
		_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: amount}, TokenIn: weth})
		assert.ErrorIs(t, err, ErrInvalidAmount, "in 2^%d", bits)
	}
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: usdt, Amount: new(big.Int).Lsh(big.NewInt(1), 300)}, TokenIn: weth})
	assert.ErrorIs(t, err, ErrInvalidAmount)
}
