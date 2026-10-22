package pools

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	v21VectorSpacing int32 = 4096
)

var v21L66 = new(int256.Int).Lsh(int256.NewInt(1), 66)

type v21Position struct {
	lower, upper int32
	liquidity    *int256.Int
}

// v21Positions are the positions minted by the contract tests that produced each vector; the vectors carry the pool
// state but not the initialized ticks.
func v21Positions(t *testing.T, name string) []v21Position {
	const s = v21VectorSpacing
	wide := v21Position{-40 * s, 40 * s, v21L66}
	switch name {
	case "inc_exact_in_cut_bridge_rate", "dec_toward_then_away_exact_in", "inc_exact_out_after_anchor_update",
		"dec_exact_out_partial", "inc_exact_in_partial_mid_segment":
		return []v21Position{wide, {3 * s, 7 * s, v21L66}, {-7 * s, -3 * s, v21L66}}
	case "exact_tick_state_dec_from_tick_T", "inc_through_T_toward_uncut", "dec_onto_T_tick_T_minus_1",
		"exact_tick_state_inc_from_tick_T_minus_1", "dec_onto_T_again", "fee_only_1_wei_exact_in_at_T":
		return []v21Position{wide, {3 * s, 7 * s, v21L66}}
	case "bridge_25pct_exact_in", "bridge_25pct_exact_out_dec", "bridge_1ulp_exact_in":
		return []v21Position{wide, {2 * s, 3 * s, v21L66}}
	case "empty_words_skip0", "return_toward_skip0", "empty_words_skip3":
		return []v21Position{{-s, s, v21L66}, {1000 * s, 1001 * s, v21L66}}
	case "cap_tail_to_max":
		return []v21Position{
			{-20_000 * s, 20_000 * s, int256.NewInt(1 << 40)},
			{3000 * s, 3001 * s, int256.NewInt(1_000_000_000)},
		}
	case "segexp2_offgrid_ticks_inc":
		return []v21Position{wide, {3 * s, 7 * s, new(int256.Int).Add(v21L66, v21L66)}, {8 * s, 9 * s, v21L66}}
	case "spacing3000_negative_anchor_dec", "spacing3000_negative_anchor_inc":
		return []v21Position{{-200 * 3000, 100 * 3000, v21L66}, {-50 * 3000, -30 * 3000, v21L66}}
	}
	t.Fatalf("no position layout for %s", name)
	return nil
}

func v21Ticks(t *testing.T, name string) []Tick {
	net := map[int32]*int256.Int{}
	for _, p := range v21Positions(t, name) {
		for _, side := range []struct {
			tick int32
			neg  bool
		}{{p.lower, false}, {p.upper, true}} {
			if net[side.tick] == nil {
				net[side.tick] = new(int256.Int)
			}
			if side.neg {
				net[side.tick].Sub(net[side.tick], p.liquidity)
			} else {
				net[side.tick].Add(net[side.tick], p.liquidity)
			}
		}
	}
	ticks := make([]Tick, 0, len(net))
	for number, delta := range net {
		ticks = append(ticks, Tick{Number: number, LiquidityDelta: delta})
	}
	slices.SortFunc(ticks, func(a, b Tick) int { return int(a.Number) - int(b.Number) })
	return ticks
}

type v21Num string

func (n *v21Num) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*n = v21Num(s)
		return nil
	}
	*n = v21Num(b)
	return nil
}

func (n v21Num) u256() *uint256.Int { return uint256.MustFromDecimal(string(n)) }
func (n v21Num) i256() *int256.Int  { return int256.MustFromDec(string(n)) }
func (n v21Num) i64() int64 {
	v, err := strconv.ParseInt(string(n), 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}
func (n v21Num) u64() uint64 {
	v, err := strconv.ParseUint(string(n), 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

type (
	v21CoreState struct {
		SqrtRatio v21Num `json:"sqrtRatio"`
		Tick      v21Num `json:"tick"`
		Liquidity v21Num `json:"liquidity"`
	}
	v21Call struct {
		Limit     v21Num       `json:"limit"`
		Amount    v21Num       `json:"amount"`
		Kind      string       `json:"kind"`
		Surcharge v21Num       `json:"surcharge"`
		Before    v21CoreState `json:"before"`
		After     v21CoreState `json:"after"`
		Delta0    v21Num       `json:"delta0"`
		Delta1    v21Num       `json:"delta1"`
		Fee0      v21Num       `json:"fee0"`
		Fee1      v21Num       `json:"fee1"`
	}
	v21Vector struct {
		Name string `json:"name"`
		Cfg  struct {
			Tau        v21Num `json:"tau"`
			SlopeK     v21Num `json:"slopeK"`
			SegExp     v21Num `json:"segExp"`
			JLin       v21Num `json:"jLin"`
			MaxFee     v21Num `json:"maxFee"`
			ClampTicks v21Num `json:"clampTicks"`
			MGate      v21Num `json:"mGate"`
		} `json:"cfg"`
		Pool struct {
			Fee         v21Num `json:"fee"`
			TickSpacing v21Num `json:"tickSpacing"`
		} `json:"pool"`
		AnchorX16 v21Num `json:"anchorX16"`
		Params    struct {
			IsToken1       bool   `json:"isToken1"`
			Amount         v21Num `json:"amount"`
			SqrtRatioLimit v21Num `json:"sqrtRatioLimit"`
			SkipAhead      v21Num `json:"skipAhead"`
		} `json:"params"`
		Before v21CoreState `json:"before"`
		After  v21CoreState `json:"after"`
		Total  struct {
			Delta0 v21Num `json:"delta0"`
			Delta1 v21Num `json:"delta1"`
		} `json:"total"`
		Calls []v21Call `json:"calls"`
	}
	v21Anchor struct {
		LastUpdateTime v21Num `json:"lastUpdateTime"`
		LRefTime       v21Num `json:"lRefTime"`
		LastPosTime    v21Num `json:"lastPosTime"`
		LRefBits       v21Num `json:"lRefBits"`
		SnapBits       v21Num `json:"snapBits"`
		LRefRaiseTime  v21Num `json:"lRefRaiseTime"`
		AnchorX16      v21Num `json:"anchorX16"`
	}
)

func (a v21Anchor) anchor() MevCaptureV21Anchor {
	return MevCaptureV21Anchor{
		LastUpdateTime: uint32(a.LastUpdateTime.u64()),
		LRefTime:       uint32(a.LRefTime.u64()),
		LastPosTime:    uint32(a.LastPosTime.u64()),
		LRefBits:       uint8(a.LRefBits.u64()),
		SnapBits:       uint8(a.SnapBits.u64()),
		LRefRaiseTime:  uint32(a.LRefRaiseTime.u64()),
		AnchorX16:      a.AnchorX16.i64(),
	}
}

func (s v21CoreState) core() mevCaptureV21CoreState {
	var c mevCaptureV21CoreState
	c.sqrtRatio.Set(SqrtRatioFromRaw(s.SqrtRatio.u256()))
	c.liquidity.Set(s.Liquidity.u256())
	c.tick = int32(s.Tick.i64())
	return c
}

func readV21Fixture(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile("testdata/mevcapture-v21/" + name)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, v))
}

func v21PoolKey(fee uint64, tickSpacing uint32) *ConcentratedPoolKey {
	return NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		NewPoolConfig(common.HexToAddress("0x00000000000000000000000000000000000000ee"), fee,
			NewConcentratedPoolTypeConfig(tickSpacing)),
	)
}

func v21VectorPool(t *testing.T, v *v21Vector) *MevCaptureV21Pool {
	config := MevCaptureV21Config{
		HalfLife:   uint32(v.Cfg.Tau.u64()),
		SlopeK:     uint8(v.Cfg.SlopeK.u64()),
		SegmentExp: uint8(v.Cfg.SegExp.u64()),
		JLin:       uint8(v.Cfg.JLin.u64()),
		MaxFee:     v.Cfg.MaxFee.u64(),
		ClampTicks: uint32(v.Cfg.ClampTicks.u64()),
		MGate:      uint8(v.Cfg.MGate.u64()),
	}
	core := v.Before.core()
	ticks := v21Ticks(t, v.Name)

	// the rebuilt ticks must reproduce the traced active liquidity
	var active uint256.Int
	for _, tick := range ticks {
		if tick.Number <= core.tick {
			require.NoError(t, addLiquidityDelta(&active, tick.LiquidityDelta, false))
		}
	}
	require.True(t, active.Eq(&core.liquidity), v.Name)

	pool, err := NewMevCaptureV21Pool(
		v21PoolKey(v.Pool.Fee.u64(), uint32(v.Pool.TickSpacing.u64())),
		NewMevCaptureV21PoolState(
			&MevCaptureV21PoolSwapState{
				SqrtRatio: core.sqrtRatio.Clone(),
				Liquidity: core.liquidity.Clone(),
				Tick:      core.tick,
				Anchor:    MevCaptureV21Anchor{LastUpdateTime: 1, LRefTime: 1, AnchorX16: int64(core.tick) << 16},
			},
			config,
			ticks,
			[2]int32{math.MinTick, math.MaxTick},
		),
	)
	require.NoError(t, err, v.Name)
	return pool
}

func requireCoreState(t *testing.T, expected v21CoreState, actual *mevCaptureV21CoreState, msg string) {
	t.Helper()
	require.Equal(t, expected.SqrtRatio.u256().Dec(), SqrtRatioToRaw(&actual.sqrtRatio).Dec(), msg+": sqrtRatio")
	require.Equal(t, int32(expected.Tick.i64()), actual.tick, msg+": tick")
	require.Equal(t, expected.Liquidity.u256().Dec(), actual.liquidity.Dec(), msg+": liquidity")
}

func TestMevCaptureV21ReplaysEveryContractVectorExactly(t *testing.T) {
	t.Parallel()

	var fixture struct {
		Vectors []v21Vector `json:"vectors"`
	}
	readV21Fixture(t, "quoter-vectors.json", &fixture)
	require.Len(t, fixture.Vectors, 21)

	callsChecked := 0
	var exactIn, exactOut, token0, token1, skip int
	for i := range fixture.Vectors {
		v := &fixture.Vectors[i]
		pool := v21VectorPool(t, v)
		params := &MevCaptureV21SwapParams{
			IsToken1:       v.Params.IsToken1,
			Amount:         v.Params.Amount.i256(),
			SqrtRatioLimit: SqrtRatioFromRaw(v.Params.SqrtRatioLimit.u256()),
			SkipAhead:      uint32(v.Params.SkipAhead.u64()),
		}
		if params.Amount.Sign() < 0 {
			exactOut++
		} else {
			exactIn++
		}
		if params.IsToken1 {
			token1++
		} else {
			token0++
		}
		if params.SkipAhead != 0 {
			skip++
		}

		core := v.Before.core()
		swap, err := pool.SwapThroughSegments(&core, params, v.AnchorX16.i64(), true)
		require.NoError(t, err, v.Name)
		require.Len(t, swap.Calls, len(v.Calls), "%s: call count", v.Name)

		var surcharges uint256.Int
		for j, c := range swap.Calls {
			e := v.Calls[j]
			msg := fmt.Sprintf("%s call %d", v.Name, j)
			require.Equal(t, e.Limit.u256().Dec(), SqrtRatioToRaw(&c.SqrtRatioLimit).Dec(), msg+": limit")
			require.Equal(t, e.Amount.i256().Dec(), c.Amount.Dec(), msg+": amount")
			require.Equal(t, e.Kind, c.Kind.String(), msg+": kind")
			require.Equal(t, e.Surcharge.u64(), c.Surcharge, msg+": surcharge")
			requireCoreState(t, e.Before, &c.Before, msg+": before")
			requireCoreState(t, e.After, &c.After, msg+": after")
			require.Equal(t, e.Delta0.i256().Dec(), c.Delta0.Dec(), msg+": delta0")
			require.Equal(t, e.Delta1.i256().Dec(), c.Delta1.Dec(), msg+": delta1")
			require.Equal(t, e.Fee0.u256().Dec(), c.Surcharge0.Dec(), msg+": fee0")
			require.Equal(t, e.Fee1.u256().Dec(), c.Surcharge1.Dec(), msg+": fee1")
			surcharges.Add(&surcharges, &c.Surcharge0)
			surcharges.Add(&surcharges, &c.Surcharge1)
			callsChecked++
		}
		require.Equal(t, v.Total.Delta0.i256().Dec(), swap.Delta0.Dec(), v.Name+": total0")
		require.Equal(t, v.Total.Delta1.i256().Dec(), swap.Delta1.Dec(), v.Name+": total1")
		requireCoreState(t, v.After, &swap.StateAfter, v.Name+": after")
		var total uint256.Int
		total.Add(&swap.Surcharge0, &swap.Surcharge1)
		require.Equal(t, surcharges.Dec(), total.Dec(), v.Name)
		require.Equal(t, uint32(len(v.Calls)), swap.Resources.CoreCalls, v.Name)
	}

	require.Equal(t, 266, callsChecked)
	require.True(t, exactIn > 0 && exactOut > 0 && token0 > 0 && token1 > 0 && skip > 0)
}

func TestMevCaptureV21SurchargesOnlyOnChargedCallsOfTheCalculatedToken(t *testing.T) {
	t.Parallel()

	var fixture struct {
		Vectors []v21Vector `json:"vectors"`
	}
	readV21Fixture(t, "quoter-vectors.json", &fixture)
	for _, v := range fixture.Vectors {
		for _, c := range v.Calls {
			fee0, fee1 := c.Fee0.u256(), c.Fee1.u256()
			if c.Kind != "charged" {
				require.True(t, fee0.IsZero() && fee1.IsZero(), v.Name)
			}
			if v.Params.IsToken1 {
				require.True(t, fee1.IsZero(), v.Name)
			} else {
				require.True(t, fee0.IsZero(), v.Name)
			}
		}
	}
}

func TestMevCaptureV21PredSuccCrossRegions(t *testing.T) {
	t.Parallel()

	for _, raw := range []*uint256.Int{
		new(uint256.Int).Or(new(uint256.Int).Lsh(big256.U1, 94), new(uint256.Int).Lsh(big256.U1, 62)),
		new(uint256.Int).Or(new(uint256.Int).Lsh(uint256.NewInt(2), 94), new(uint256.Int).Lsh(big256.U1, 62)),
		new(uint256.Int).Or(new(uint256.Int).Lsh(uint256.NewInt(3), 94), new(uint256.Int).Lsh(big256.U1, 62)),
		uint256.NewInt(1 << 63),
	} {
		x := SqrtRatioFromRaw(raw)
		pred, succ := PredSqrtRatio(x), SuccSqrtRatio(x)
		require.True(t, SuccSqrtRatio(pred).Eq(x))
		require.True(t, pred.Lt(x) && math.IsValidSqrtRatio(pred))
		require.True(t, succ.Gt(x) && math.IsValidSqrtRatio(succ))
		require.True(t, SqrtRatioToRaw(x).Eq(raw))
	}
}

// The fixtures are from the variant of the extension that floors the anchor movement allowance; this extension rounds
// it up, so the cases bound by a fractional floored allowance move exactly one more unit.
func TestMevCaptureV21AnchorUpdatesAllSevenFields(t *testing.T) {
	t.Parallel()

	var fixture struct {
		Config struct {
			HalfLife   v21Num `json:"halfLife"`
			SlopeK     v21Num `json:"slopeK"`
			SegmentExp v21Num `json:"segmentExp"`
			JLin       v21Num `json:"jLin"`
			MaxFee     v21Num `json:"maxFee"`
			ClampTicks v21Num `json:"clampTicks"`
			MGate      v21Num `json:"mGate"`
		} `json:"config"`
		AnchorUpdates []struct {
			StateIn   v21Anchor `json:"stateIn"`
			Tick      v21Num    `json:"tick"`
			Liquidity v21Num    `json:"liquidity"`
			Timestamp v21Num    `json:"timestamp"`
			Passed    bool      `json:"passed"`
			LObs      v21Num    `json:"lObs"`
			ERef      v21Num    `json:"eRef"`
			Expected  v21Anchor `json:"expected"`
		} `json:"anchorUpdates"`
	}
	readV21Fixture(t, "anchor-updates.json", &fixture)
	require.Len(t, fixture.AnchorUpdates, 120)

	c := fixture.Config
	config := MevCaptureV21Config{
		HalfLife:   uint32(c.HalfLife.u64()),
		SlopeK:     uint8(c.SlopeK.u64()),
		SegmentExp: uint8(c.SegmentExp.u64()),
		JLin:       uint8(c.JLin.u64()),
		MaxFee:     c.MaxFee.u64() << 48,
		ClampTicks: uint32(c.ClampTicks.u64()),
		MGate:      uint8(c.MGate.u64()),
	}
	require.NoError(t, config.Validate())

	halfLife, clamp := uint64(config.HalfLife), uint64(config.ClampTicks)
	roundedUp := 0
	for i, tc := range fixture.AnchorUpdates {
		stateIn, expected := tc.StateIn.anchor(), tc.Expected.anchor()
		now := uint32(tc.Timestamp.u64())
		dt := min(uint64(now-stateIn.LastUpdateTime), halfLife)
		floored := (clamp << 16) * dt / halfLife
		nextMove := expected.AnchorX16 - stateIn.AnchorX16
		absMove := uint64(max(nextMove, -nextMove))
		if tc.Passed && (clamp<<16)*dt%halfLife != 0 && absMove == floored {
			if nextMove > 0 {
				expected.AnchorX16++
			} else if nextMove < 0 {
				expected.AnchorX16--
			}
			roundedUp++
		}

		state := stateIn
		gate := config.BeforeSwap(&state, int32(tc.Tick.i64()), tc.Liquidity.u256(), now)
		require.NotNil(t, gate, "case %d", i)
		require.Equal(t, tc.Passed, gate.Passed, "case %d", i)
		require.Equal(t, uint8(tc.LObs.u64()), gate.LObs, "case %d", i)
		require.Equal(t, uint8(tc.ERef.u64()), gate.ERef, "case %d", i)
		require.Equal(t, expected, state, "case %d", i)
	}
	require.Positive(t, roundedUp, "no fixture exercises the rounded-up allowance")
}

func v21ClampConfig(halfLife, clampTicks uint32) MevCaptureV21Config {
	return MevCaptureV21Config{
		HalfLife:   halfLife,
		SlopeK:     4,
		SegmentExp: 0,
		JLin:       16,
		MaxFee:     1 << 63,
		ClampTicks: clampTicks,
		MGate:      3,
	}
}

var v21L66Liquidity = new(uint256.Int).Lsh(big256.U1, 66)

func TestMevCaptureV21PerSecondUpdatesCannotFreezeAnchor(t *testing.T) {
	t.Parallel()

	config := v21ClampConfig(1<<20, 1)
	var touched MevCaptureV21Anchor
	for i := uint32(1); i <= 32; i++ {
		require.NotNil(t, config.BeforeSwap(&touched, 1000, v21L66Liquidity, i))
		require.Equal(t, i, touched.LastUpdateTime)
		require.Equal(t, int64(i), touched.AnchorX16)
	}
	var single MevCaptureV21Anchor
	config.BeforeSwap(&single, 1000, v21L66Liquidity, 32)
	require.Equal(t, int64(2), single.AnchorX16)
}

func TestMevCaptureV21DustAllowanceRoundsUp(t *testing.T) {
	t.Parallel()

	config := v21ClampConfig(120, 2500)
	var state MevCaptureV21Anchor
	config.BeforeSwap(&state, 1_000_000, v21L66Liquidity, 7)
	require.Equal(t, ((int64(2500)<<16)*7+119)/120, state.AnchorX16)
}

func TestMevCaptureV21FrequentUpdatesNeverSlowAnchor(t *testing.T) {
	t.Parallel()

	for _, tau := range []uint32{1, 2, 3, 7, 120, 1000, 65_537, 1 << 20} {
		for _, clamp := range []uint32{1, 2, 5, 2500, 4096} {
			for _, n := range []uint32{1, 2, 17, 48} {
				config := v21ClampConfig(tau, clamp)
				tick := int32(70*clamp + 128)
				units := uint64(clamp) << 16
				tau64 := uint64(tau)

				var touched MevCaptureV21Anchor
				for i := uint32(1); i <= n; i++ {
					config.BeforeSwap(&touched, tick, v21L66Liquidity, i)
				}
				var single MevCaptureV21Anchor
				config.BeforeSwap(&single, tick, v21L66Liquidity, n)

				touchedMove, singleMove := uint64(touched.AnchorX16), uint64(single.AnchorX16)
				require.Equal(t, uint64(n)*((units+tau64-1)/tau64), touchedMove, "%d %d %d", tau, clamp, n)
				require.Equal(t, (units*min(uint64(n), tau64)+tau64-1)/tau64, singleMove)
				require.GreaterOrEqual(t, touchedMove, singleMove)
				require.LessOrEqual(t, touchedMove*tau64, uint64(n)*units+uint64(n)*tau64)
			}
		}
	}
}

func v21TestPool(t *testing.T) *MevCaptureV21Pool {
	var fixture struct {
		Vectors []v21Vector `json:"vectors"`
	}
	readV21Fixture(t, "quoter-vectors.json", &fixture)
	idx := slices.IndexFunc(fixture.Vectors, func(v v21Vector) bool { return v.Name == "inc_exact_in_cut_bridge_rate" })
	require.GreaterOrEqual(t, idx, 0)
	return v21VectorPool(t, &fixture.Vectors[idx])
}

func TestMevCaptureV21QuoteRunsAnchorUpdateThenSegments(t *testing.T) {
	t.Parallel()

	pool := v21TestPool(t)
	amount := uint256.MustFromDecimal("1000000000000000000")

	same, err := pool.quoteAt(amount, true, 1)
	require.NoError(t, err)

	core := mevCaptureV21CoreState{tick: pool.Tick}
	core.sqrtRatio.Set(pool.SqrtRatio)
	core.liquidity.Set(pool.Liquidity)
	direct, err := pool.SwapThroughSegments(&core, &MevCaptureV21SwapParams{
		IsToken1:       true,
		Amount:         int256.MustFromDec("1000000000000000000"),
		SqrtRatioLimit: math.MaxSqrtRatio,
	}, 0, false)
	require.NoError(t, err)
	require.Equal(t, direct.Delta1.Dec(), same.ConsumedAmount.Dec())
	var out uint256.Int
	out.Neg((*uint256.Int)(&direct.Delta0))
	require.Equal(t, out.Dec(), same.CalculatedAmount.Dec())
	require.False(t, direct.Surcharge0.IsZero())
	require.Equal(t, mevCaptureV21Gas(&direct.Resources), same.Gas)
	require.Equal(t, uint32(1), same.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState).Anchor.LastUpdateTime)

	later, err := pool.quoteAt(amount, true, 13)
	require.NoError(t, err)
	stateAfter := later.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState)
	require.Equal(t, uint32(13), stateAfter.Anchor.LastUpdateTime)
	require.Equal(t, uint32(13), stateAfter.swapTime)
	require.Greater(t, later.Gas, same.Gas)
	require.Equal(t, pool.key.Config.Extension, *later.SwapInfo.Forward)
	require.Zero(t, later.SwapInfo.SkipAhead)
	require.Equal(t, math.FixedSqrtRatioToFloat(math.MaxSqrtRatio, false), later.SwapInfo.PriceLimit)
}

func TestMevCaptureV21SwapStateCarriesTimestampAndAnchor(t *testing.T) {
	t.Parallel()

	pool := v21TestPool(t)
	amount := uint256.MustFromDecimal("100000000000000000")

	first, err := pool.quoteAt(amount, false, 100)
	require.NoError(t, err)

	cloned := pool.CloneSwapStateOnly().(*MevCaptureV21Pool)
	cloned.SetSwapState(first.SwapInfo.SwapStateAfter)

	// the clone runs at the timestamp of the swap it applied and does not update the anchor again
	second, err := cloned.Quote(amount, false)
	require.NoError(t, err)
	secondState := second.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState)
	require.Equal(t, uint32(100), secondState.swapTime)
	require.Equal(t, uint32(100), secondState.Anchor.LastUpdateTime)
	require.Equal(t, first.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState).Anchor.AnchorX16,
		secondState.Anchor.AnchorX16)

	direct, err := cloned.quoteAt(amount, false, 100)
	require.NoError(t, err)
	require.Equal(t, direct.CalculatedAmount, second.CalculatedAmount)

	// the original pool is unchanged
	again, err := pool.quoteAt(amount, false, 100)
	require.NoError(t, err)
	require.Equal(t, first.CalculatedAmount, again.CalculatedAmount)
	require.Equal(t, first.Gas, again.Gas)
	require.Zero(t, pool.swapTime)
}

func TestMevCaptureV21QuoteStopsAtTheKnownTickRange(t *testing.T) {
	t.Parallel()

	pool := v21TestPool(t)
	state := *pool.MevCaptureV21PoolState
	state.TickBounds = [2]int32{-5 * v21VectorSpacing, 5 * v21VectorSpacing}
	state.SortedTicks = slices.DeleteFunc(slices.Clone(state.SortedTicks), func(tick Tick) bool {
		return tick.Number < state.TickBounds[0] || tick.Number > state.TickBounds[1]
	})
	bounded, err := NewMevCaptureV21Pool(pool.key, &state)
	require.NoError(t, err)

	amount := uint256.MustFromDecimal("1000000000000000000000000")
	quote, err := bounded.quoteAt(amount, true, 1)
	require.NoError(t, err)
	require.True(t, quote.ConsumedAmount.Lt(amount))
	after := quote.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState)
	require.Equal(t, math.ToSqrtRatio(state.TickBounds[1]), after.SqrtRatio)
	require.Equal(t, math.FixedSqrtRatioToFloat(math.ToSqrtRatio(state.TickBounds[1]), false), quote.SwapInfo.PriceLimit)

	quote, err = bounded.quoteAt(amount, false, 1)
	require.NoError(t, err)
	require.True(t, quote.ConsumedAmount.Lt(amount))
	after = quote.SwapInfo.SwapStateAfter.(*MevCaptureV21PoolSwapState)
	require.Equal(t, math.ToSqrtRatio(state.TickBounds[0]), after.SqrtRatio)
}
