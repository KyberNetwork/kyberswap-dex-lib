package ladder

import (
	"errors"
	"math"
	"math/big"
	"sort"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrOverflow       = errors.New("ladder: uint256 overflow")
	ErrNegativeResult = errors.New("ladder: negative result")
)

// decimalsSweepSteps is how many multiplier steps on each side of
// 10^decimals BuildDecimalsSweep spans (7 magnitude anchors total: center
// plus decimalsSweepSteps steps down and up).
const decimalsSweepSteps = 3

// decimalsSweepStepMultiplier is the per-step multiplier between adjacent
// magnitude anchors -- wider than a plain order-of-magnitude (10x) step so
// the sweep still brackets tokens whose real tradeable range sits well away
// from 10^decimals.
const decimalsSweepStepMultiplier = 32

// BuildSamplePointsNU256 is BuildSamplePointsN on uint256. A nil or zero
// reserve yields nil.
func BuildSamplePointsNU256(reserve *uint256.Int, n int) []*uint256.Int {
	return buildSamplePointsFromReserveU256(reserve, geometricBpsRange(sampleBpsMin, sampleBpsMax, n))
}

// BuildSamplePointsFromU256 is BuildSamplePointsFrom on uint256. Each point is
// floor(floor(x*10000/9900)*b/10000); the 257-bit first quotient is split by
// splitReserveU256 so the two floors stay exact. ErrOverflow is never returned
// for an in-range x.
func BuildSamplePointsFromU256(nearCapacityAmount *uint256.Int, n int) ([]*uint256.Int, error) {
	if nearCapacityAmount == nil || nearCapacityAmount.IsZero() {
		return nil, nil
	}
	hi, lo := splitReserveU256(nearCapacityAmount)
	bps := dgeoBps(n)
	points := make([]*uint256.Int, 0, len(bps))
	for _, b := range bps {
		var pt uint256.Int
		pointFromSplitU256(&pt, hi, &lo, uint64(b))
		if !pt.IsZero() {
			points = append(points, &pt)
		}
	}
	sort.Slice(points, func(a, b int) bool { return points[a].Lt(points[b]) })
	return dedupSortedU256(points), nil
}

// splitReserveU256 returns q = floor(x*10000/9900) as (hi, lo): hi is bit 256
// (0 or 1) and lo the low 256 bits. MulDivOverflow's flag is exactly hi.
func splitReserveU256(x *uint256.Int) (hi uint64, lo uint256.Int) {
	var overflow bool
	_, overflow = lo.MulDivOverflow(x, uint256.NewInt(10000), uint256.NewInt(sampleBpsMax))
	if overflow {
		hi = 1
	}
	return hi, lo
}

// pointFromSplitU256 sets z = floor((hi*2^256 + lo)*b/10000). For hi=1 it uses
// 2^256 = Q*10000 + R, so the carry from (lo*b mod 10000 + b*R) is added.
func pointFromSplitU256(z *uint256.Int, hi uint64, lo *uint256.Int, b uint64) {
	var bi uint256.Int
	bi.SetUint64(b)
	if hi == 0 {
		big256.MulDivDown(z, lo, &bi, big256.UBasisPoint)
		return
	}
	var t, rem, carry uint256.Int
	t.MulDivOverflow(lo, &bi, big256.UBasisPoint)
	rem.MulMod(lo, &bi, big256.UBasisPoint)
	carry.Mul(&bi, pow2256ModBasisPoint)
	carry.Add(&carry, &rem)
	carry.Div(&carry, big256.UBasisPoint)
	z.Mul(&bi, pow2256DivBasisPoint)
	z.Add(z, &t)
	z.Add(z, &carry)
}

// pow2256DivBasisPoint and pow2256ModBasisPoint are 2^256 / 10000 and 2^256 % 10000.
// SetAllOne is 2^256-1, so the remainder is bumped by one (2^256-1 has R-1 here).
var (
	pow2256DivBasisPoint = new(uint256.Int).Div(new(uint256.Int).SetAllOne(), big256.UBasisPoint)
	pow2256ModBasisPoint = new(uint256.Int).Add(new(uint256.Int).Mod(new(uint256.Int).SetAllOne(), big256.UBasisPoint), uint256.NewInt(1))
)

func buildSamplePointsFromReserveU256(reserve *uint256.Int, bps []int) []*uint256.Int {
	if reserve == nil || reserve.IsZero() {
		return nil
	}

	points := make([]*uint256.Int, 0, len(bps))
	for _, b := range bps {
		var pt uint256.Int
		big256.MulDivDown(&pt, reserve, uint256.NewInt(uint64(b)), big256.UBasisPoint)
		if !pt.IsZero() {
			points = append(points, &pt)
		}
	}

	sort.Slice(points, func(a, b int) bool { return points[a].Lt(points[b]) })
	return dedupSortedU256(points)
}

// EstimateNearCapacityAmountU256 is EstimateNearCapacityAmount on uint256.
// Returns nil when the previous ladder shows no rate drop.
func EstimateNearCapacityAmountU256(prevLadder []Point, prevOutputReserve, currentOutputReserve *uint256.Int) *uint256.Int {
	nearCapIn, found := DepletionAmountIn(prevLadder)
	if !found {
		return nil
	}
	return scaleByOutputReserveChangeU256(nearCapIn, prevOutputReserve, currentOutputReserve)
}

// EstimateFarthestProbedAmountU256 is EstimateFarthestProbedAmount on uint256.
// Returns nil if prevLadder is empty.
func EstimateFarthestProbedAmountU256(prevLadder []Point, prevOutputReserve, currentOutputReserve *uint256.Int) *uint256.Int {
	if len(prevLadder) == 0 {
		return nil
	}
	return scaleByOutputReserveChangeU256(prevLadder[len(prevLadder)-1].AmountIn(), prevOutputReserve, currentOutputReserve)
}

// scaleByOutputReserveChangeU256 projects amountIn by the output-reserve
// ratio. The ratio stays float64; the result truncates toward zero.
func scaleByOutputReserveChangeU256(amountIn float64, prevOutputReserve, currentOutputReserve *uint256.Int) *uint256.Int {
	if amountIn <= 0 || prevOutputReserve == nil || prevOutputReserve.IsZero() ||
		currentOutputReserve == nil || currentOutputReserve.IsZero() {
		return nil
	}

	estimate := amountIn * (currentOutputReserve.Float64() / prevOutputReserve.Float64())
	if estimate <= 0 {
		return nil
	}

	result, _ := big.NewFloat(estimate).Int(nil)
	out, overflow := uint256.FromBig(result)
	if overflow {
		return nil
	}
	return out
}

// SamplePointsU256 is SamplePoints on uint256. It returns ErrOverflow when a
// guided grid cannot be scaled into 256 bits.
func SamplePointsU256(p entity.Pool, dir int, currentInputReserve, currentOutputReserve *uint256.Int) ([]*uint256.Int, error) {
	prevLadder, prevOutputReserve, ok := prevLadderAndReserveU256(p, dir)
	if !ok {
		return firstProbePointsU256(p, dir), nil
	}

	nearCap := EstimateNearCapacityAmountU256(prevLadder, prevOutputReserve, currentOutputReserve)
	if nearCap == nil {
		return firstProbePointsU256(p, dir), nil
	}

	grid, err := BuildSamplePointsFromU256(nearCap, SampleSize)
	if err != nil {
		return nil, err
	}
	points, err := withGrowthCanariesU256(grid, nearCap, currentInputReserve)
	if err != nil {
		return nil, err
	}
	if farthest := EstimateFarthestProbedAmountU256(prevLadder, prevOutputReserve, currentOutputReserve); farthest != nil {
		points = appendClampedU256(points, farthest, currentInputReserve)
	}
	return points, nil
}

func firstProbePointsU256(p entity.Pool, dir int) []*uint256.Int {
	if dir < 0 || dir >= len(p.Tokens) {
		return nil
	}
	return BuildDecimalsSweepU256(p.Tokens[dir].Decimals)
}

// BuildDecimalsSweepU256 is BuildDecimalsSweep on uint256. The sweep's
// upper bound 10^decimals * 32^3 overflows only past decimals=72, where it
// returns nil.
func BuildDecimalsSweepU256(decimals uint8) []*uint256.Int {
	return BuildDecimalsSweepNU256(decimals, SampleSize)
}

// BuildDecimalsSweepNU256 is BuildDecimalsSweepN on uint256.
func BuildDecimalsSweepNU256(decimals uint8, n int) []*uint256.Int {
	var center, factor, lo, hi uint256.Int
	if !powOverflowU256(&center, 10, uint64(decimals)) {
		return nil
	}
	powOverflowU256(&factor, decimalsSweepStepMultiplier, decimalsSweepSteps)
	lo.Div(&center, &factor)
	if lo.IsZero() {
		lo.SetOne()
	}
	if _, overflow := hi.MulOverflow(&center, &factor); overflow {
		return nil
	}
	return dedupSortedU256(geometricUintRangeU256(&lo, &hi, n))
}

// geometricUintRangeU256 returns n geometrically-spaced points from lo to hi
// inclusive, sorted ascending. Interior points truncate to integers, and
// both endpoints are pinned exactly.
func geometricUintRangeU256(lo, hi *uint256.Int, n int) []*uint256.Int {
	if n <= 1 {
		return []*uint256.Int{new(uint256.Int).Set(hi)}
	}
	loF := lo.Float64()
	hiF := hi.Float64()
	ratio := math.Pow(hiF/loF, 1/float64(n-1))
	points := make([]*uint256.Int, n)
	v := loF
	for i := range n - 1 {
		pt, _ := big.NewFloat(v).Int(nil)
		points[i], _ = uint256.FromBig(pt)
		v *= ratio
	}
	points[0] = new(uint256.Int).Set(lo)
	points[n-1] = new(uint256.Int).Set(hi)
	return points
}

// withGrowthCanariesU256 appends canaryGrowthMultiples of nearCap, returning
// ErrOverflow if a canary does not fit in 256 bits.
func withGrowthCanariesU256(points []*uint256.Int, nearCap, currentInputReserve *uint256.Int) ([]*uint256.Int, error) {
	for _, m := range canaryGrowthMultiples {
		var amount uint256.Int
		if _, overflow := amount.MulOverflow(nearCap, uint256.NewInt(uint64(m))); overflow {
			return nil, ErrOverflow
		}
		points = appendClampedU256(points, &amount, currentInputReserve)
	}
	return points, nil
}

// appendClampedU256 is appendClamped on uint256. A zero currentInputReserve
// means no cap, matching appendClamped.
func appendClampedU256(points []*uint256.Int, amount, currentInputReserve *uint256.Int) []*uint256.Int {
	if currentInputReserve != nil && !currentInputReserve.IsZero() && amount.Gt(currentInputReserve) {
		amount = currentInputReserve
	}
	if !amount.IsZero() {
		points = append(points, amount)
	}
	sort.Slice(points, func(a, b int) bool { return points[a].Lt(points[b]) })
	return dedupSortedU256(points)
}

// powOverflowU256 sets z = base^exp, reporting false on 256-bit overflow.
func powOverflowU256(z *uint256.Int, base, exp uint64) bool {
	z.SetOne()
	b := uint256.NewInt(base)
	for range exp {
		if _, overflow := z.MulOverflow(z, b); overflow {
			return false
		}
	}
	return true
}

// prevLadderAndReserveU256 is prevLadderAndReserve with the output reserve as
// uint256. ok is false on a malformed or missing reserve string.
func prevLadderAndReserveU256(p entity.Pool, dir int) (prevLadder []Point, prevOutputReserve *uint256.Int, ok bool) {
	if p.Extra == "" || len(p.Reserves) != 2 {
		return nil, nil, false
	}

	var prevExtra Extra
	if err := json.Unmarshal([]byte(p.Extra), &prevExtra); err != nil {
		return nil, nil, false
	}

	prevOutputReserve, err := uint256.FromDecimal(p.Reserves[1-dir])
	if err != nil {
		return nil, nil, false
	}
	return prevExtra.Ladders[dir], prevOutputReserve, true
}

func dedupSortedU256(sorted []*uint256.Int) []*uint256.Int {
	if len(sorted) <= 1 {
		return sorted
	}
	result := sorted[:1]
	for _, v := range sorted[1:] {
		if !v.Eq(result[len(result)-1]) {
			result = append(result, v)
		}
	}
	return result
}

// CollectLadderU256 is CollectLadder on uint256 amounts. Points are still
// float64, so amounts convert at this boundary.
func CollectLadderU256(points []*uint256.Int, results []*uint256.Int) []Point {
	pts := make([]Point, 0, len(points))
	for i, amt := range points {
		if i >= len(results) {
			break
		}
		out := results[i]
		if out == nil || out.IsZero() {
			continue
		}
		pts = append(pts, Point{amt.Float64(), out.Float64()})
	}
	return pts
}
