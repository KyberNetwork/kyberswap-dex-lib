package ladder

import (
	"math"
	"math/big"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// SampleSize is the default number of reserve-fraction probe points built by
// BuildSamplePoints / BuildSamplePointsFrom.
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

// BuildSamplePoints returns a sorted, deduplicated grid of SampleSize probe
// amounts, geometrically spaced between sampleBpsMin and sampleBpsMax and
// scaled by reserve. Geometric (constant-ratio) spacing keeps every gap the
// same relative size, unlike a hand-picked list where an arbitrary large gap
// can sit exactly under a real trade size and blow up the interpolation
// error (see Spline).
//
// This is used for a pool's first probe, where there's no previous ladder
// yet to say where the pool's real depletion knee sits -- plain,
// symmetric-in-log-space spacing makes no assumption about which end of the
// range that knee will fall near. Once a previous cycle's ladder is
// available, EstimateNearCapacityAmount / BuildSamplePointsFrom re-anchor
// the range at the pool's actual depletion point and switch to the
// dgeo-spaced grid, which concentrates far more points right at that
// reserve cap.
func BuildSamplePoints(reserve *big.Int) []*big.Int {
	return BuildSamplePointsN(reserve, SampleSize)
}

// BuildSamplePointsN is like BuildSamplePoints, but for a grid of n probe
// amounts instead of SampleSize. Use a smaller n where quoting is expensive.
func BuildSamplePointsN(reserve *big.Int, n int) []*big.Int {
	return buildSamplePointsFromReserve(reserve, geometricBpsRange(sampleBpsMin, sampleBpsMax, n))
}

func buildSamplePointsFromReserve(reserve *big.Int, bps []int) []*big.Int {
	if reserve == nil || reserve.Sign() <= 0 {
		return nil
	}

	points := make([]*big.Int, 0, len(bps))
	for _, b := range bps {
		pt := new(big.Int).Mul(reserve, big.NewInt(int64(b)))
		pt.Div(pt, bignumber.BasisPoint)
		if pt.Sign() > 0 {
			points = append(points, pt)
		}
	}

	sort.Slice(points, func(a, b int) bool { return points[a].Cmp(points[b]) < 0 })
	return dedupSorted(points)
}

// rateDropFraction is how far a ladder's marginal rate of return (the
// amountOut/amountIn secant of one probe-to-probe segment) is allowed to
// fall below the best marginal rate seen so far before DepletionAmountIn
// considers the pool to have started depleting.
const rateDropFraction = 0.90

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

// EstimateNearCapacityAmount estimates, with no extra on-chain calls, the
// amountIn just past where the previous cycle's ladder showed the pool's
// marginal rate of return start dropping (see DepletionAmountIn), using the
// previous cycle's ladder and output-side reserve as a guide.
//
// reserve0 alone can badly overstate the tradeable range for an imbalanced
// pool (see BuildSamplePoints's doc): sampling up to 99% of reserve0 assumes
// the other side has enough inventory to pay it out, which isn't always
// true, and the real depletion point can sit at a small fraction of the
// reserve for a badly imbalanced pool. The previous ladder already recorded
// where the rate started dropping last cycle (in terms of that cycle's
// reserve); scaling it by how much the output-side reserve has since changed
// re-estimates it for this cycle without probing anything new.
//
// Returns nil if the previous ladder never showed a rate drop (no knee
// observed last time), in which case the caller should fall back to its
// default reserve-based basis.
func EstimateNearCapacityAmount(prevLadder []Point, prevOutputReserve, currentOutputReserve *big.Int) *big.Int {
	if prevOutputReserve == nil || prevOutputReserve.Sign() <= 0 ||
		currentOutputReserve == nil || currentOutputReserve.Sign() <= 0 {
		return nil
	}

	nearCapIn, found := DepletionAmountIn(prevLadder)
	if !found {
		return nil
	}

	prevReserveF, _ := prevOutputReserve.Float64()
	currentReserveF, _ := currentOutputReserve.Float64()
	estimate := nearCapIn * (currentReserveF / prevReserveF)
	if estimate <= 0 {
		return nil
	}

	result, _ := big.NewFloat(estimate).Int(nil)
	return result
}

// BuildSamplePointsFrom is like BuildSamplePointsN, but scales a
// double-geometric ("dgeo") grid -- geometric from sampleBpsMin up to the
// midpoint (sampleBpsMin+sampleBpsMax)/2, mirrored around that midpoint for
// the top half -- from nearCapacityAmount (see EstimateNearCapacityAmount)
// instead of directly off a reserve: nearCapacityAmount is treated as the
// sampleBpsMax point, and the equivalent "reserve" is backed out so the
// existing bps math applies unchanged.
//
// A plain geometric progression spends half its points on the bottom half
// of the range even though, once re-anchored at a real depletion point this
// way, the real knee is almost always right at the top -- mirroring the
// bottom half's spacing onto the top half concentrates far more points
// there instead.
func BuildSamplePointsFrom(nearCapacityAmount *big.Int, n int) []*big.Int {
	if nearCapacityAmount == nil || nearCapacityAmount.Sign() <= 0 {
		return nil
	}
	reserve := new(big.Int).Mul(nearCapacityAmount, bignumber.BasisPoint)
	reserve.Div(reserve, big.NewInt(sampleBpsMax))
	return buildSamplePointsFromReserve(reserve, dgeoBps(n))
}

func dedupSorted(sorted []*big.Int) []*big.Int {
	if len(sorted) <= 1 {
		return sorted
	}
	result := sorted[:1]
	for _, v := range sorted[1:] {
		if v.Cmp(result[len(result)-1]) != 0 {
			result = append(result, v)
		}
	}
	return result
}

// CollectLadder pairs each probe amount with its quoted output, converting to
// float64 and dropping any point that reverted or returned zero.
func CollectLadder(points []*big.Int, results []*big.Int) []Point {
	pts := make([]Point, 0, len(points))
	for i, amt := range points {
		out := results[i]
		if out == nil || out.Sign() <= 0 {
			continue
		}
		amtF, _ := amt.Float64()
		outF, _ := out.Float64()
		pts = append(pts, Point{amtF, outF})
	}
	return pts
}
