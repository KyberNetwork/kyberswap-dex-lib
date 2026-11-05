package dexv2

import (
	"errors"
	"testing"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func maxUint256() *uint256.Int { return new(uint256.Int).SetAllOne() }

func TestSwapPriceLimitValidation(t *testing.T) {
	sim, _ := snapshotSimulator(t)
	pool := sim.V3Pool
	amount := int256.NewInt(1_000_000)
	current := pool.SqrtRatioX96
	below := new(uint256.Int).SubUint64(current, 1)
	above := new(uint256.Int).AddUint64(current, 1)

	_, err := pool.swap(true, amount, new(uint256.Int).SubUint64(v3Utils.MinSqrtRatioU256, 1))
	require.ErrorIs(t, err, v3Entities.ErrSqrtPriceLimitX96TooLow)
	_, err = pool.swap(true, amount, above)
	require.ErrorIs(t, err, v3Entities.ErrSqrtPriceLimitX96TooHigh)
	_, err = pool.swap(false, amount, new(uint256.Int).AddUint64(v3Utils.MaxSqrtRatioU256, 1))
	require.ErrorIs(t, err, v3Entities.ErrSqrtPriceLimitX96TooHigh)
	_, err = pool.swap(false, amount, below)
	require.ErrorIs(t, err, v3Entities.ErrSqrtPriceLimitX96TooLow)
	// A nil limit defaults to the extreme price in the swap direction.
	_, err = pool.swap(true, amount, nil)
	require.NoError(t, err)
	_, err = pool.swap(false, amount, nil)
	require.NoError(t, err)
}

type failingTicks struct {
	v3Entities.TickDataProvider
	next    func(tick int, lte bool, spacing int) (int, bool, error)
	getTick error
}

func (f failingTicks) NextInitializedTickWithinOneWord(tick int, lte bool, spacing int) (int, bool, error) {
	if f.next != nil {
		return f.next(tick, lte, spacing)
	}
	return f.TickDataProvider.NextInitializedTickWithinOneWord(tick, lte, spacing)
}

func (f failingTicks) GetTick(tick int) (v3Entities.Tick, error) {
	if f.getTick != nil {
		return v3Entities.Tick{}, f.getTick
	}
	return f.TickDataProvider.GetTick(tick)
}

func TestSwapTickProviderFailures(t *testing.T) {
	boom := errors.New("boom")
	amount := int256.NewInt(1_000_000_000)
	for name, mk := range map[string]func(base v3Entities.TickDataProvider) v3Entities.TickDataProvider{
		"next tick lookup": func(b v3Entities.TickDataProvider) v3Entities.TickDataProvider {
			return failingTicks{TickDataProvider: b, next: func(int, bool, int) (int, bool, error) { return 0, false, boom }}
		},
		"tick data": func(b v3Entities.TickDataProvider) v3Entities.TickDataProvider {
			// Report an initialized tick right above the price so the swap crosses it.
			return failingTicks{TickDataProvider: b, getTick: boom, next: func(tick int, _ bool, _ int) (int, bool, error) { return tick + 1, true, nil }}
		},
	} {
		sim, _ := snapshotSimulator(t)
		pool := *sim.V3Pool
		pool.TickDataProvider = mk(pool.TickDataProvider)
		// 50bn crosses the next initialized tick, so both providers are consulted.
		_, err := pool.swap(false, int256.NewInt(0).SetUint64(5_000_000_000_000), nil)
		require.ErrorIs(t, err, boom, name)
	}
	for _, tick := range []int{MIN_TICK, MAX_TICK, MIN_TICK - 1, MAX_TICK + 1} {
		sim, _ := snapshotSimulator(t)
		pool := *sim.V3Pool
		pool.TickDataProvider = failingTicks{TickDataProvider: pool.TickDataProvider,
			next: func(int, bool, int) (int, bool, error) { return tick, false, nil }}
		_, err := pool.swap(true, amount, nil)
		require.ErrorIs(t, err, ErrNextTickOutOfBounds, "tick %d", tick)
	}
}

func TestSwapStepOutputBound(t *testing.T) {
	// Liquidity this deep makes a single step output exceed the 86-bit module bound, in both the
	// constant-fee and the dynamic-fee loops.
	huge := new(uint256.Int).SetOne()
	huge.Lsh(huge, 120)
	for name, mk := range map[string]func() *PoolSimulator{
		"constant fee": func() *PoolSimulator { s, _ := snapshotSimulator(t); return s },
		"dynamic fee":  func() *PoolSimulator { s, _ := dynamicSnapshot(t, 0); return s },
	} {
		sim := mk()
		pool := *sim.V3Pool
		pool.Liquidity = huge
		amount := new(int256.Int).SetUint64(1)
		amount.Lsh(amount, 200)
		_, err := pool.swap(false, amount, nil)
		require.ErrorIs(t, err, ErrOverflow, name)
	}
}

func TestDynamicActiveLiquidityBound(t *testing.T) {
	sim, _ := dynamicSnapshot(t, 7)
	pool := *sim.V3Pool
	// More than 102 bits of active liquidity cannot be stored in the packed variables.
	pool.Liquidity = new(uint256.Int).Lsh(uint256.NewInt(1), 103)
	limit := new(uint256.Int).SubUint64(pool.SqrtRatioX96, 1)
	amount := new(int256.Int).Lsh(int256.NewInt(1), 100)
	_, err := pool.swap(true, amount, limit)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestNetSwapInStepClearsWhenFeeExceedsOutput(t *testing.T) {
	var net, protocol, lp uint256.Int
	gross := uint256.NewInt(1)
	netSwapInStep(&net, &protocol, &lp, gross, uint256.NewInt(4095), uint256.NewInt(100))
	require.True(t, net.IsZero(), "protocol charge consumes the whole output")
	require.Equal(t, uint64(1), protocol.Uint64())
	require.True(t, lp.IsZero())

	net, protocol, lp = uint256.Int{}, uint256.Int{}, uint256.Int{}
	gross = uint256.NewInt(1)
	netSwapInStep(&net, &protocol, &lp, gross, uint256.NewInt(0), uint256.NewInt(65535))
	require.True(t, net.IsZero(), "LP charge consumes the whole output")
	require.Equal(t, uint64(1), lp.Uint64())
}

func TestDynamicFeePrimitivesRejectOutOfRange(t *testing.T) {
	var dst uint256.Int
	require.ErrorIs(t, priceAtSqrt(&dst, maxUint256()), ErrOverflow)

	// Impact is capped at 100% and the kink price is clamped to the supported range.
	var kink, sqrt uint256.Int
	anchor := new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	require.NoError(t, dynamicKink(&kink, &sqrt, anchor, 10_000_000, false))
	require.Equal(t, new(uint256.Int).Mul(anchor, uint256.NewInt(2)).Dec(), kink.Dec(), "capped at +100%")
	require.NoError(t, dynamicKink(&kink, &sqrt, uint256.NewInt(1), 10_000_000, true))
	require.Equal(t, uint64(1_350_587), kink.Uint64(), "clamped to the minimum price")
	require.NoError(t, dynamicKink(&kink, &sqrt, new(uint256.Int).Lsh(uint256.NewInt(1), 172), 0, false))
	require.Equal(t, maxDynamicPriceX96.Dec(), kink.Dec(), "clamped to the maximum price")
	// Above 2^160 the square root is taken at lower precision and scaled back up.
	big160 := new(uint256.Int).Lsh(uint256.NewInt(1), 165)
	require.NoError(t, dynamicKink(&kink, &sqrt, big160, 0, false))
	require.False(t, sqrt.IsZero())
	require.ErrorIs(t, dynamicKink(&kink, &sqrt, maxUint256(), 1_000_000, false), ErrOverflow)
}

func TestPrepareAndFinishDynamicFeeRejectInvalidState(t *testing.T) {
	sim, _ := dynamicSnapshot(t, 7)
	packed := sim.V3Pool.dynamicPacked
	sqrt := new(uint256.Int).Set(sim.V3Pool.SqrtRatioX96)
	ts := sim.extra.SnapshotTimestamp

	_, err := prepareDynamicFee(&packed, maxUint256(), true, ts)
	require.ErrorIs(t, err, ErrOverflow)

	// A stored negative impact of 100% or more has no anchor price.
	bad := packed
	putPackedBits(&bad, 208, 1, 0)
	putPackedBits(&bad, 209, 20, 1_000_000)
	putPackedBits(&bad, 229, 15, ts&32767)
	putPackedBits(&bad, 244, 12, 4000)
	_, err = prepareDynamicFee(&bad, sqrt, true, ts)
	require.ErrorIs(t, err, ErrUnsupportedDynamicFee)

	d, err := prepareDynamicFee(&packed, sqrt, true, ts)
	require.NoError(t, err)
	_, err = finishDynamicFee(&d, maxUint256())
	require.ErrorIs(t, err, ErrOverflow)
	// Ending 2x above the anchor is a >=100% price impact.
	far := new(uint256.Int).Mul(sqrt, uint256.NewInt(3))
	_, err = finishDynamicFee(&d, far)
	require.ErrorIs(t, err, ErrUnsupportedDynamicFee)
}

func TestDynamicSwapStepAccumulatorOverflow(t *testing.T) {
	sim, _ := dynamicSnapshot(t, 7)
	packed := sim.V3Pool.dynamicPacked
	current := new(uint256.Int).Set(sim.V3Pool.SqrtRatioX96)
	d, err := prepareDynamicFee(&packed, current, true, sim.extra.SnapshotTimestamp)
	require.NoError(t, err)
	var target uint256.Int
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(sim.V3Pool.TickCurrent-40, &target))
	liquidity := uint256.NewInt(1_000_000_000_000_000)
	protocolFee := uint256.NewInt(4095)

	run := func(preset func(acc *[5]uint256.Int)) error {
		var acc [5]uint256.Int // amountIn, netOut, grossOut, protocolOut, lpOut
		preset(&acc)
		var next uint256.Int
		remaining := int256.NewInt(5_000_000_000)
		return dynamicSwapStep(&d, true, current, &target, liquidity, remaining, protocolFee,
			&next, &acc[0], &acc[1], &acc[2], &acc[3], &acc[4])
	}
	require.NoError(t, run(func(*[5]uint256.Int) {}))
	for i, name := range []string{"amount in", "net out", "gross out", "protocol fee", "LP fee"} {
		err := run(func(acc *[5]uint256.Int) { acc[i].SetAllOne() })
		require.ErrorIs(t, err, ErrOverflow, name)
	}
	// A step output beyond the 86-bit module bound is rejected before any fee is taken.
	deep := new(uint256.Int).Lsh(uint256.NewInt(1), 120)
	var next, a, b, c, dd, e uint256.Int
	err = dynamicSwapStep(&d, true, current, &target, deep, new(int256.Int).Lsh(int256.NewInt(1), 200), protocolFee, &next, &a, &b, &c, &dd, &e)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestPrepareDynamicFeeZeroAnchor(t *testing.T) {
	sim, _ := dynamicSnapshot(t, 7)
	packed := sim.V3Pool.dynamicPacked
	// A price of zero (sqrt too small for Q96 squaring) leaves no anchor to measure impact from.
	_, err := prepareDynamicFee(&packed, uint256.NewInt(1), true, sim.extra.SnapshotTimestamp)
	require.ErrorIs(t, err, ErrUnsupportedDynamicFee)
}

// Starting between the min-fee and max-fee kinks, a dust input produces no output, so the
// dynamic zone charges no fee at all.
func TestDynamicSwapStepMiddleZoneWithoutOutput(t *testing.T) {
	sim, _ := dynamicSnapshot(t, 7)
	packed := sim.V3Pool.dynamicPacked
	ts := sim.extra.SnapshotTimestamp
	putPackedBits(&packed, 208, 1, 0)
	putPackedBits(&packed, 209, 20, 5000) // price 0.5% below the zero-impact anchor
	putPackedBits(&packed, 229, 15, ts&32767)
	putPackedBits(&packed, 244, 12, 600)
	current := new(uint256.Int).Set(sim.V3Pool.SqrtRatioX96)
	d, err := prepareDynamicFee(&packed, current, true, ts)
	require.NoError(t, err)
	var target uint256.Int
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(sim.V3Pool.TickCurrent-40, &target))
	var next, amountIn, net, gross, protocol, lp uint256.Int
	err = dynamicSwapStep(&d, true, current, &target, uint256.NewInt(1_000_000_000), int256.NewInt(1), uint256.NewInt(0),
		&next, &amountIn, &net, &gross, &protocol, &lp)
	require.NoError(t, err)
	t.Logf("net=%s gross=%s protocol=%s lp=%s", &net, &gross, &protocol, &lp)
}

func TestCalculateVarsEighteenDecimalCode(t *testing.T) {
	// Storage code 15 stands for 18 decimals on either token.
	cfg := packExchangePrices(0, 0, 0, 0, 0, precision1e12, precision1e12, 0, 0)
	v := new(uint256.Int).Lsh(uint256.NewInt(15), BITS_DEX_V2_VARIABLES2_TOKEN_0_DECIMALS)
	c, err := _calculateVars(v.ToBig(), cfg, cfg, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000_000), c.Token0DenominatorPrecision.Int64())
}
