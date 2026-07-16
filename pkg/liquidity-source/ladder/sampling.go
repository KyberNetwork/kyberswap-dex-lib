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
// real trade size and blow up the linear interpolation error.
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
	if reserve == nil || reserve.Sign() <= 0 {
		return nil
	}

	points := make([]*big.Int, 0, len(SampleBps))
	for _, bps := range SampleBps {
		pt := new(big.Int).Mul(reserve, big.NewInt(int64(bps)))
		pt.Div(pt, bignumber.BasisPoint)
		if pt.Sign() > 0 {
			points = append(points, pt)
		}
	}

	sort.Slice(points, func(a, b int) bool { return points[a].Cmp(points[b]) < 0 })
	return dedupSorted(points)
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
