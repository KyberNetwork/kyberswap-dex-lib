// Package propamm provides shared helpers for propAMM-family integrations (wasabi-prop, kipseli-prop, etc.).
//
// PropAMM pools behave like orderbook + oracle hybrids: near-constant rate within a valid
// amountIn range, with hard boundaries where output drops to zero. The helpers here implement
// an incremental sample strategy that discovers and tracks this valid range efficiently.
package propamm

import (
	"math"
	"math/big"
	"sort"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	SampleSize       = 15
	NumLowerProbes   = 2
	NumUpperProbes   = 3
	NumDensePoints   = SampleSize - NumLowerProbes - NumUpperProbes
	ColdRefinePoints = 5
)

// BuildQueryPoints picks sample amountIn values depending on available history:
//   - Cold start: 10^k grid spanning token decimals range.
//   - Incremental: dense log-spaced in [prevMin, prevMax] + boundary probes outside.
func BuildQueryPoints(decimals uint8, prevMin, prevMax *big.Int) []*big.Int {
	if prevMin == nil || prevMin.Sign() <= 0 || prevMax == nil || prevMax.Sign() <= 0 || prevMax.Cmp(prevMin) <= 0 {
		return ColdStartGrid(decimals)
	}
	return lo.Flatten([][]*big.Int{
		lowerProbes(prevMin),
		LogSpaced(prevMin, prevMax, NumDensePoints),
		upperProbes(prevMax),
	})
}

// ColdStartGrid returns SampleSize points: 10^start, 10^(start+1), ..., centered around token decimals.
func ColdStartGrid(decimals uint8) []*big.Int {
	start := lo.Ternary(decimals < SampleSize/2, 0, decimals-SampleSize/2)
	return lo.Times(SampleSize, func(k int) *big.Int { return bignumber.TenPowInt(start + uint8(k)) })
}

// LogSpaced returns n points logarithmically spaced between low and high (inclusive).
func LogSpaced(low, high *big.Int, n int) []*big.Int {
	l, h := math.Log10(bigIntToFloat64(low)), math.Log10(bigIntToFloat64(high))
	return lo.Times(n, func(k int) *big.Int {
		if k == 0 {
			return low
		}
		if k == n-1 {
			return high
		}
		r, _ := new(big.Float).SetFloat64(math.Pow(10, l+(h-l)*float64(k)/float64(n-1))).Int(nil)
		return r
	})
}

// lowerProbes: prevMin/2, prevMin/5 — detect if valid range expanded downward.
func lowerProbes(v *big.Int) []*big.Int {
	return lo.Map([]int64{2, 5}, func(d int64, _ int) *big.Int {
		if amt := new(big.Int).Div(v, big.NewInt(d)); amt.Sign() > 0 {
			return amt
		}
		return big.NewInt(1)
	})
}

// upperProbes: prevMax*1.5, *2, *4 — detect if cap shifted upward.
func upperProbes(v *big.Int) []*big.Int {
	return lo.Map([][2]int64{{3, 2}, {2, 1}, {4, 1}}, func(f [2]int64, _ int) *big.Int {
		amt := new(big.Int).Mul(v, big.NewInt(f[0]))
		return amt.Div(amt, big.NewInt(f[1]))
	})
}

func bigIntToFloat64(x *big.Int) float64 {
	f, _ := new(big.Float).SetInt(x).Float64()
	return f
}

// CleanSamples filters out<=0, sorts by amountIn, deduplicates.
func CleanSamples(s [][2]*big.Int) [][2]*big.Int {
	s = lo.Filter(s, func(v [2]*big.Int, _ int) bool {
		return v[0] != nil && v[1] != nil && v[1].Sign() > 0
	})
	sort.Slice(s, func(a, b int) bool { return s[a][0].Cmp(s[b][0]) < 0 })
	return lo.UniqBy(s, func(v [2]*big.Int) string { return v[0].String() })
}

// ValidRangeFromSamples returns [min, max] amountIn from previous run's samples.
// Returns nil,nil on cold start (no previous data).
func ValidRangeFromSamples(allSamples [][][2]*big.Int, dir int) (prevMin, prevMax *big.Int) {
	if dir >= len(allSamples) || len(allSamples[dir]) == 0 {
		return nil, nil
	}
	s := allSamples[dir]
	if first, last := s[0][0], s[len(s)-1][0]; first != nil && first.Sign() > 0 && last != nil && last.Sign() > 0 {
		return first, last
	}
	return nil, nil
}

// FindCapBoundary finds the transition where amountOut drops to zero.
// Returns (highest amountIn with out>0, lowest amountIn with out==0 above it).
func FindCapBoundary(samples [][2]*big.Int) (capLower, capUpper *big.Int) {
	for _, s := range samples {
		if s[0] != nil && s[1] != nil && s[1].Sign() > 0 && (capLower == nil || s[0].Cmp(capLower) > 0) {
			capLower = s[0]
		}
	}
	if capLower == nil {
		return
	}
	for _, s := range samples {
		if s[0] != nil && s[1] != nil && s[0].Cmp(capLower) > 0 && s[1].Sign() == 0 &&
			(capUpper == nil || s[0].Cmp(capUpper) < 0) {
			capUpper = s[0]
		}
	}
	return
}

// RefineCapPoints returns evenly spaced points between capLower and capUpper for cap boundary refinement.
func RefineCapPoints(capLower, capUpper *big.Int) []*big.Int {
	if capLower == nil || capUpper == nil || capUpper.Cmp(capLower) <= 0 {
		return nil
	}
	gap, den := new(big.Int).Sub(capUpper, capLower), big.NewInt(ColdRefinePoints+1)
	return lo.FilterMap(
		lo.Times(ColdRefinePoints, func(j int) *big.Int {
			amt := new(big.Int).Mul(gap, big.NewInt(int64(j+1)))
			return amt.Div(amt, den).Add(amt, capLower)
		}),
		func(amt *big.Int, _ int) (*big.Int, bool) {
			return amt, amt.Cmp(capLower) > 0 && amt.Cmp(capUpper) < 0
		},
	)
}

// ApplyBuffer scales down amountOut by buffer/BasisPoint as a safety margin.
func ApplyBuffer(samples [][][2]*big.Int, buffer int64) {
	if buffer <= 0 {
		return
	}
	buf := big.NewInt(buffer)
	for i := range samples {
		for j := range samples[i] {
			if samples[i][j][1] != nil {
				samples[i][j][1].Mul(samples[i][j][1], buf).Div(samples[i][j][1], bignumber.BasisPoint)
			}
		}
	}
}
