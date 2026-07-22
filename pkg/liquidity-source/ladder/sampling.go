package ladder

import (
	"math"
	"math/big"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// SampleSize is the number of reserve-fraction probe points in SampleBps.
const SampleSize = 16

// sampleBpsMin/Max bound the reserve-fraction probe grid. 9900bps is as
// close to the full reserve as this pattern of pool typically quotes without
// reverting.
const (
	sampleBpsMin = 10
	sampleBpsMax = 9900
)

// SampleBps is a geometric sequence of SampleSize reserve-fraction levels
// from sampleBpsMin to sampleBpsMax. Geometric (constant-ratio) spacing keeps
// every gap the same relative size, unlike a hand-picked list where an
// arbitrary large gap (e.g. the old 50->250bps jump) can sit exactly under a
// real trade size and blow up the interpolation error (see Spline).
var SampleBps = geometricBps(SampleSize)

func geometricBps(n int) []int {
	if n <= 1 {
		return []int{sampleBpsMax}
	}
	ratio := math.Pow(float64(sampleBpsMax)/float64(sampleBpsMin), 1/float64(n-1))
	bps := make([]int, n)
	v := float64(sampleBpsMin)
	for i := range n - 1 {
		bps[i] = int(math.Round(v))
		v *= ratio
	}
	bps[n-1] = sampleBpsMax // pin the exact endpoint against float drift
	return bps
}

// BuildSamplePoints returns a sorted, deduplicated grid of probe amounts: one
// point per SampleBps entry scaled by reserve.
func BuildSamplePoints(reserve *big.Int) []*big.Int {
	return BuildSamplePointsN(reserve, SampleSize)
}

// BuildSamplePointsN returns a sorted, deduplicated grid of n probe amounts,
// geometrically spaced between sampleBpsMin and sampleBpsMax and scaled by
// reserve. Use a smaller n where quoting is expensive.
func BuildSamplePointsN(reserve *big.Int, n int) []*big.Int {
	return buildSamplePointsFromReserve(reserve, n)
}

func buildSamplePointsFromReserve(reserve *big.Int, n int) []*big.Int {
	if reserve == nil || reserve.Sign() <= 0 {
		return nil
	}

	bps := geometricBps(n)
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

// nearCapacityFraction is the target fraction of the output-side reserve
// that EstimateNearCapacityAmount looks for in the previous cycle's ladder:
// the smallest previously-sampled amountIn whose amountOut already reached
// this fraction of that cycle's reserve.
const nearCapacityFraction = 0.95

// EstimateNearCapacityAmount estimates, with no extra on-chain calls, the
// amountIn that would deplete ~nearCapacityFraction of the CURRENT
// output-side reserve, using the previous cycle's ladder and output-side
// reserve as a guide.
//
// reserve0 alone can badly overstate the tradeable range for an imbalanced
// pool (see BuildSamplePoints's doc): sampling up to 99% of reserve0 assumes
// the other side has enough inventory to pay it out, which isn't always
// true, and can be sparse or even revert entirely past the pool's real
// depletion point. The previous ladder already recorded where that
// depletion point was last cycle (in terms of that cycle's reserve); scaling
// it by how much the output-side reserve has since changed re-estimates it
// for this cycle without probing anything new.
//
// Returns nil if the previous ladder never got within
// nearCapacityFraction of its cycle's reserve (no knee observed last time),
// in which case the caller should fall back to its default reserve-based
// basis.
func EstimateNearCapacityAmount(prevLadder []Point, prevOutputReserve, currentOutputReserve *big.Int) *big.Int {
	if prevOutputReserve == nil || prevOutputReserve.Sign() <= 0 ||
		currentOutputReserve == nil || currentOutputReserve.Sign() <= 0 {
		return nil
	}

	prevReserveF, _ := prevOutputReserve.Float64()
	target := prevReserveF * nearCapacityFraction

	var nearCapIn float64
	found := false
	for _, p := range prevLadder {
		if p.AmountOut() >= target {
			nearCapIn = p.AmountIn()
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	currentReserveF, _ := currentOutputReserve.Float64()
	estimate := nearCapIn * (currentReserveF / prevReserveF)
	if estimate <= 0 {
		return nil
	}

	result, _ := big.NewFloat(estimate).Int(nil)
	return result
}

// BuildSamplePointsFrom is like BuildSamplePointsN, but scales the
// geometric grid from nearCapacityAmount (see EstimateNearCapacityAmount)
// instead of directly off a reserve: nearCapacityAmount is treated as the
// sampleBpsMax point, and the equivalent "reserve" is backed out so the
// existing bps math applies unchanged.
func BuildSamplePointsFrom(nearCapacityAmount *big.Int, n int) []*big.Int {
	if nearCapacityAmount == nil || nearCapacityAmount.Sign() <= 0 {
		return nil
	}
	reserve := new(big.Int).Mul(nearCapacityAmount, bignumber.BasisPoint)
	reserve.Div(reserve, big.NewInt(sampleBpsMax))
	return buildSamplePointsFromReserve(reserve, n)
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
