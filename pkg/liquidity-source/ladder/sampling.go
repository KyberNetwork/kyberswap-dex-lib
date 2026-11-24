package ladder

import (
	"math"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// SampleSize is the default number of reserve-fraction probe points built by
// BuildSamplePointsN / BuildSamplePointsFrom / BuildDecimalsSweep.
const SampleSize = 16

// sampleBpsMin/Max bound the reserve-fraction probe grid. 9900bps is as
// close to the full reserve as this pattern of pool typically quotes without
// reverting.
const (
	sampleBpsMin = 10
	sampleBpsMax = 9900
)

func dgeoBps(n int) []int {
	if n < 4 {
		return geometricBpsRange(sampleBpsMin, sampleBpsMax, n)
	}

	mid := (sampleBpsMin + sampleBpsMax) / 2

	if n%2 == 1 {
		// Odd n: the midpoint is one of the n points, shared by both
		// halves -- the first half's own geometric spacing (pinned to
		// exactly reach mid) mirrors cleanly onto the second half.
		n1 := n/2 + 1
		firstHalf := geometricBpsRange(sampleBpsMin, mid, n1)
		bps := make([]int, 0, n)
		bps = append(bps, firstHalf...)
		for i := len(firstHalf) - 2; i >= 0; i-- {
			bps = append(bps, min(2*mid-firstHalf[i], sampleBpsMax))
		}
		return bps
	}

	// Even n: there's no single shared midpoint -- the two innermost points
	// straddle mid symmetrically instead, so the first half doesn't reach
	// mid exactly. Solving for the ratio that keeps the gap across the
	// middle perfectly in step with the rest of the progression is a
	// transcendental equation; using (n+1)/2 "steps" as the exponent
	// (instead of n/2, which leaves an enormous gap right at the seam) is a
	// close practical approximation, within ~10% of seamless at
	// SampleSize=16, the only size actually used in this codebase.
	half := n / 2
	ratio := math.Pow(float64(sampleBpsMax)/float64(sampleBpsMin), 1/(float64(n+1)/2))
	firstHalf := make([]int, half)
	v := float64(sampleBpsMin)
	for i := range half {
		firstHalf[i] = int(math.Round(v))
		v *= ratio
	}
	firstHalf[0] = sampleBpsMin

	bps := make([]int, 0, n)
	bps = append(bps, firstHalf...)
	for i := half - 1; i >= 0; i-- {
		bps = append(bps, min(2*mid-firstHalf[i], sampleBpsMax))
	}
	bps[len(bps)-1] = sampleBpsMax // pin the exact endpoint against float drift
	return bps
}

// geometricBpsRange returns n geometrically-spaced points from lo to hi
// inclusive (both endpoints included, first pinned to lo, last pinned to hi
// against float drift).
func geometricBpsRange(lo, hi, n int) []int {
	if n <= 1 {
		return []int{hi}
	}
	ratio := math.Pow(float64(hi)/float64(lo), 1/float64(n-1))
	bps := make([]int, n)
	v := float64(lo)
	for i := range n - 1 {
		bps[i] = int(math.Round(v))
		v *= ratio
	}
	bps[n-1] = hi
	return bps
}

// BuildSamplePointsN returns a sorted, deduplicated grid of n probe amounts,
// geometrically spaced between sampleBpsMin and sampleBpsMax and scaled by
// reserve. Amounts that do not fit in uint256 (or a negative reserve) map to
// nil, matching the legacy embedder contract (see BuildSamplePointsNU256).
func BuildSamplePointsN(reserve *big.Int, n int) []*big.Int {
	u, ok := fromBigOrNil(reserve)
	if !ok {
		return nil
	}
	return toBigs(BuildSamplePointsNU256(u, n))
}

// rateDropFraction is how far a ladder's marginal rate of return (the
// amountOut/amountIn secant of one probe-to-probe segment) is allowed to
// fall below the best marginal rate seen so far before DepletionAmountIn
// considers the pool to have started depleting. A plain constant-product
// pool's marginal rate declines continuously even with no real depletion at
// all, and a coarse geometric grid's last segment can cross a 10% relative
// drop purely from that ordinary curvature -- 20% comfortably clears that
// false-positive band while still catching real on-chain depletion knees,
// which collapse far harder than this (verified against captured ladders).
const rateDropFraction = 0.80

// DepletionAmountIn scans a ladder (sorted ascending by AmountIn, as probed)
// for the first point whose segment's marginal rate of return has dropped
// to rateDropFraction of the best marginal rate seen earlier in the ladder,
// and returns that point's AmountIn.
//
// A pool's marginal rate declines gently and continuously from ordinary
// slippage long before it's anywhere near depleted, then falls off a cliff
// right at the depletion knee -- rateDropFraction (10%) is well above the
// gentle, gradual decline and only trips on that cliff. Past that point the
// pool is giving increasingly poor returns per unit in, so there's no value
// in sampling further out.
//
// Returns 0, false if the ladder never shows that much of a drop (fewer than
// two points, or still entirely within its "good rate" zone).
func DepletionAmountIn(ladder []Point) (float64, bool) {
	if len(ladder) < 2 {
		return 0, false
	}

	bestRate := ladder[0].AmountOut() / ladder[0].AmountIn()
	prevIn, prevOut := ladder[0].AmountIn(), ladder[0].AmountOut()

	for _, p := range ladder[1:] {
		rate := (p.AmountOut() - prevOut) / (p.AmountIn() - prevIn)
		if rate <= bestRate*rateDropFraction {
			return p.AmountIn(), true
		}
		if rate > bestRate {
			bestRate = rate
		}
		prevIn, prevOut = p.AmountIn(), p.AmountOut()
	}
	return 0, false
}

// EstimateNearCapacityAmount is EstimateNearCapacityAmountU256 on big.Int.
// Out-of-range reserves map to nil.
func EstimateNearCapacityAmount(prevLadder []Point, prevOutputReserve, currentOutputReserve *big.Int) *big.Int {
	prev, okPrev := fromBigOrNil(prevOutputReserve)
	cur, okCur := fromBigOrNil(currentOutputReserve)
	if !okPrev || !okCur {
		return nil
	}
	return toBig(EstimateNearCapacityAmountU256(prevLadder, prev, cur))
}

// EstimateFarthestProbedAmount is EstimateFarthestProbedAmountU256 on big.Int.
// Out-of-range reserves map to nil.
func EstimateFarthestProbedAmount(prevLadder []Point, prevOutputReserve, currentOutputReserve *big.Int) *big.Int {
	prev, okPrev := fromBigOrNil(prevOutputReserve)
	cur, okCur := fromBigOrNil(currentOutputReserve)
	if !okPrev || !okCur {
		return nil
	}
	return toBig(EstimateFarthestProbedAmountU256(prevLadder, prev, cur))
}

// BuildSamplePointsFrom is BuildSamplePointsFromU256 on big.Int. A negative
// or out-of-range input maps to an empty grid.
func BuildSamplePointsFrom(nearCapacityAmount *big.Int, n int) []*big.Int {
	u, ok := fromBigOrNil(nearCapacityAmount)
	if !ok {
		return nil
	}
	pts, err := BuildSamplePointsFromU256(u, n)
	if err != nil {
		return nil
	}
	return toBigs(pts)
}

// SamplePoints is SamplePointsU256 on big.Int. An out-of-range input maps to
// nil.
func SamplePoints(p entity.Pool, dir int, currentInputReserve, currentOutputReserve *big.Int) []*big.Int {
	in, _ := fromBigOrNil(currentInputReserve)
	out, _ := fromBigOrNil(currentOutputReserve)
	pts, err := SamplePointsU256(p, dir, in, out)
	if err != nil {
		return nil
	}
	return toBigs(pts)
}

// BuildDecimalsSweep is BuildDecimalsSweepU256 on big.Int.
func BuildDecimalsSweep(decimals uint8) []*big.Int {
	return BuildDecimalsSweepN(decimals, SampleSize)
}

// BuildDecimalsSweepN is BuildDecimalsSweepNU256 on big.Int.
func BuildDecimalsSweepN(decimals uint8, n int) []*big.Int {
	return toBigs(BuildDecimalsSweepNU256(decimals, n))
}

// canaryGrowthMultiples scales nearCapacityAmount by each factor to produce
// extra probe points past a guided grid's own ceiling.
//
// DepletionAmountIn's rate-drop signal is a one-way ratchet on its own:
// once a cycle's guided range undershoots the real capacity, the next
// cycle's rescaled grid exhibits the same relative curvature near ITS OWN
// top and re-triggers the same "depletion" call, so the range keeps
// shrinking and never recovers -- confirmed by simulating repeated cycles
// against a plain constant-product pool with no real depletion at all, where
// the guided range converged to a small fraction of the true reserve and
// got stuck there. These canaries test past the current ceiling every
// single cycle (scaled off that ceiling, not the raw reserve, so they stay
// proportionate to what's already been observed rather than jumping
// straight back to the full unguided range), so growth is caught
// immediately instead of only on the rare cycle where the whole guided grid
// happens to come back with no rate-drop detected at all.
var canaryGrowthMultiples = []int64{2, 4}

// CollectLadder is CollectLadderU256 on big.Int. An out-of-range amount or
// output maps to a dropped point.
func CollectLadder(points []*big.Int, results []*big.Int) []Point {
	pts := make([]Point, 0, len(points))
	for i, amt := range points {
		if i >= len(results) {
			break
		}
		a, okA := fromBigOrNil(amt)
		o, okO := fromBigOrNil(results[i])
		if !okA || !okO {
			continue
		}
		pts = append(pts, CollectLadderU256([]*uint256.Int{a}, []*uint256.Int{o})...)
	}
	return pts
}

// ZeroUnquotedReserves zeroes the output-side reserve of every direction with
// no quote: an empty ladders[dir] means no token 1-dir is obtainable, so the
// pool must not report it as liquidity.
func ZeroUnquotedReserves(reserves entity.PoolReserves, ladders [2][]Point) entity.PoolReserves {
	for dir, l := range ladders {
		if len(l) == 0 && 1-dir < len(reserves) {
			reserves[1-dir] = "0"
		}
	}
	return reserves
}

// fromBigOrNil converts a big.Int to uint256. A nil or negative value maps to
// nil, and an out-of-range value reports ok=false.
func fromBigOrNil(b *big.Int) (*uint256.Int, bool) {
	if b == nil || b.Sign() < 0 {
		return nil, false
	}
	u, overflow := uint256.FromBig(b)
	return u, !overflow
}

func toBig(u *uint256.Int) *big.Int {
	if u == nil {
		return nil
	}
	return u.ToBig()
}

func toBigs(us []*uint256.Int) []*big.Int {
	if us == nil {
		return nil
	}
	out := make([]*big.Int, len(us))
	for i, u := range us {
		out[i] = u.ToBig()
	}
	return out
}
