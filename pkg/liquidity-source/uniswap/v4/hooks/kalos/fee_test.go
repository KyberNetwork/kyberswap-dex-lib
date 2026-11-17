package kalos

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// testdata/fee_vectors.json comes from KalosDynamic's own independent reference of the fee engine
// (test/reference/fee.mjs in https://github.com/fugakux/kalos-v4-hook, which its Solidity suite checks against
// 6000 cases): one case per path through the engine (window kind, pay case, peak step, move, clamped time, cap), each
//
//	[params(14), state(8), tick, timestamp, zeroForOne, fee, paid accumulator, next state(8), changed]
//
// with params in FeeParams order (bools as 0/1) and states in FeeState order; then splitFee cases
// [kind, feeSide, kalosShareBps, total, zeroForOne, hookFee, lpFee] and hookFeeOf cases [amount, h, fee].
type feeVectors struct {
	Cases   [][]json.RawMessage `json:"cases"`
	Splits  [][7]uint32         `json:"splits"`
	Amounts [][3]any            `json:"amounts"`
}

func loadVectors(t *testing.T) feeVectors {
	t.Helper()
	raw, err := os.ReadFile("testdata/fee_vectors.json")
	require.NoError(t, err)
	var v feeVectors
	require.NoError(t, json.Unmarshal(raw, &v))
	require.NotEmpty(t, v.Cases)
	return v
}

func paramsOf(a [14]uint64) FeeParams {
	return FeeParams{
		MinFeeZeroForOne: uint32(a[0]), MaxFeeZeroForOne: uint32(a[1]),
		MinFeeOneForZero: uint32(a[2]), MaxFeeOneForZero: uint32(a[3]),
		Dynamic: a[4] == 1, Directional: a[5] == 1,
		FilterPeriod: a[6], DecayPeriod: a[7], ReductionFactor: a[8], VariableFeeControl: a[9],
		MaxVolatilityAccumulator: a[10], ResetPeriod: a[11], ResetTickFilter: a[12], SameMomentSeconds: a[13],
	}
}

func stateOf(a [8]int64) FeeState {
	return FeeState{
		ReferenceTick: int32(a[0]), LastTick: int32(a[1]), Peak: int32(a[2]), VolatilityReference: uint64(a[3]),
		LastSwap: uint64(a[4]), ResetTime: uint64(a[5]), MoveTime: uint64(a[6]), PeakTime: uint64(a[7]),
	}
}

func TestComputeFee_ReferenceVectors(t *testing.T) {
	t.Parallel()
	for i, c := range loadVectors(t).Cases {
		require.Len(t, c, 9)
		var (
			p                           [14]uint64
			s, next                     [8]int64
			tick, ts, zfo, fee, va, chg int64
		)
		for j, dst := range []any{&p, &s, &tick, &ts, &zfo, &fee, &va, &next, &chg} {
			require.NoError(t, json.Unmarshal(c[j], dst), "case %d field %d", i, j)
		}
		params, state := paramsOf(p), stateOf(s)
		gotFee, gotVA, gotNext, ok := params.computeFee(state, int32(tick), uint64(ts), zfo == 1)
		require.True(t, ok, "case %d", i)
		require.Equal(t, uint32(fee), gotFee, "case %d fee", i)
		require.Equal(t, uint64(va), gotVA, "case %d accumulator", i)
		require.Equal(t, stateOf(next), gotNext, "case %d next state", i)
		require.Equal(t, chg == 1, gotNext != state, "case %d changed", i)
	}
}

func TestSplitFee_ReferenceVectors(t *testing.T) {
	t.Parallel()
	for i, c := range loadVectors(t).Splits {
		h, lp := splitFee(uint8(c[0]), uint8(c[1]), uint16(c[2]), c[3], c[4] == 1)
		require.Equal(t, c[5], h, "split %d hook fee", i)
		require.Equal(t, c[6], lp, "split %d lp fee", i)
		// The swapper pays at most the total: 1 - (1 - h)(1 - lp) <= T.
		paid := uint64(h)*pips + uint64(lp)*pips - uint64(h)*uint64(lp)
		require.LessOrEqual(t, paid, uint64(c[3])*pips, "split %d", i)
	}
}

func TestHookFeeOf_ReferenceVectors(t *testing.T) {
	t.Parallel()
	for i, c := range loadVectors(t).Amounts {
		amount := uint256.MustFromDecimal(c[0].(string))
		h := uint32(c[1].(float64))
		fee := hookFeeOf(amount, h)
		require.Equal(t, c[2].(string), fee.Dec(), "amount %d", i)

		// grossOf is the least gross whose net after the fee reaches amount.
		gross := grossOf(amount, h)
		fee = hookFeeOf(&gross, h)
		var net uint256.Int
		net.Sub(&gross, &fee)
		require.False(t, net.Lt(amount), "amount %d: grossOf too small", i)
		if !gross.IsZero() {
			var less uint256.Int
			less.SubUint64(&gross, 1)
			fee = hookFeeOf(&less, h)
			net.Sub(&less, &fee)
			require.True(t, net.Lt(amount), "amount %d: grossOf not the least", i)
		}
	}
}

// Without a swap in between, the fee never rises with time (so pricing a second early is never below the chain).
func TestComputeFee_NeverRisesWithTime(t *testing.T) {
	t.Parallel()
	for i, c := range loadVectors(t).Cases {
		var p [14]uint64
		var s [8]int64
		var tick, ts, zfo int64
		for j, dst := range []any{&p, &s, &tick, &ts, &zfo} {
			require.NoError(t, json.Unmarshal(c[j], dst))
		}
		params, state := paramsOf(p), stateOf(s)
		prev, _, _, ok := params.computeFee(state, int32(tick), uint64(ts), zfo == 1)
		require.True(t, ok)
		for dt := uint64(1); dt <= 4000; dt += 1 + dt/4 {
			fee, _, _, ok := params.computeFee(state, int32(tick), uint64(ts)+dt, zfo == 1)
			require.True(t, ok)
			require.LessOrEqual(t, fee, prev, "case %d at +%ds", i, dt)
			prev = fee
		}
	}
}

func TestComputeFee_RevertingState(t *testing.T) {
	t.Parallel()
	p := FeeParams{MinFeeZeroForOne: 500, MaxFeeZeroForOne: 5000, MinFeeOneForZero: 500, MaxFeeOneForZero: 5000,
		Dynamic: true, FilterPeriod: 10, DecayPeriod: 120, ResetPeriod: 120}
	_, _, _, ok := p.computeFee(FeeState{LastSwap: 100, ResetTime: 200}, 0, 150, true)
	require.False(t, ok, "resetTime after the swap underflows on chain")
	_, _, _, ok = p.computeFee(FeeState{LastSwap: 100, ResetTime: 100}, 0, 150, true)
	require.True(t, ok)
}

func TestVariableFee_Saturates(t *testing.T) {
	t.Parallel()
	require.Equal(t, uint64(167966)*1000*1000/1e6, variableFee(167966, 1000))
	// The largest the contract allows: below 2^71 before the division.
	want := new(big.Int).Mul(big.NewInt(16777215), new(big.Int).Exp(big.NewInt(8388607), big.NewInt(2), nil))
	require.Equal(t, want.Div(want, big.NewInt(1e6)).Uint64(), variableFee(16777215, 8388607))
	require.Equal(t, ^uint64(0), variableFee(1<<40, 1<<30))
}
