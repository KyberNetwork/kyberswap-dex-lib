// Package propamm provides shared helpers for propAMM-family integrations (wasabi-prop, kipseli-prop, etc.).
//
// PropAMM pools have near-constant rate within a valid range, with hard cap at boundaries:
//
//	amountOut
//	    │
//	    │         ┌──────────────┐
//	    │        ╱                ╲  (or plateau instead of drop)
//	    │       ╱   constant rate  ╲
//	    │      ╱                    ╲
//	    ├─────╱──────────────────────╲────── amountIn
//	    │   validMin              validMax
//
// Strategy: sample on-chain quotes at chosen amountIn points, then interpolate.
//
//	Cold start (round 1):     10^0, 10^1, ..., 10^14  — broad discovery
//	Cold start (round 2):     refine between last-ok and first-zero — narrow cap
//	Incremental:              dense in [prevMin, prevMax] + probes outside — track shifts
//
//	    ──────[===prevMin====dense====prevMax===]──────
//	      ↑probe                              probe↑
//	    prevMin/5                           prevMax*4
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

// BuildQueryPoints picks sample amountIn values:
//
//	Cold start (no history):   [10^0, 10^1, ..., 10^14]
//	Incremental (has history): [min/2, min/5] + [dense log-spaced] + [max*1.5, max*2, max*4]
//	                            └─lower─┘       └──prevMin..Max──┘   └───upper probes───┘
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

// ColdStartGrid: 15 points spanning token's magnitude range.
//
//	USDC (6 dec): 10^0, 10^1, ..., 10^14  (0.000001 USDC → 100M USDC)
//	WETH (18 dec): 10^11, 10^12, ..., 10^25
func ColdStartGrid(decimals uint8) []*big.Int {
	start := lo.Ternary(decimals < SampleSize/2, 0, decimals-SampleSize/2)
	return lo.Times(SampleSize, func(k int) *big.Int { return bignumber.TenPowInt(start + uint8(k)) })
}

// LogSpaced: n points log-distributed between low..high (inclusive endpoints).
//
//	low=100, high=10000, n=5 → [100, 316, 1000, 3162, 10000]
//	                             equal spacing in log10 scale
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

// CleanSamples: filter → sort → dedup → trim plateau.
//
//	Before: [1→100] [10→0] [5→500] [100→3000] [1000→3000] [10000→3000]
//	Filter: [1→100]        [5→500] [100→3000] [1000→3000] [10000→3000]
//	Sort:   [1→100] [5→500] [100→3000] [1000→3000] [10000→3000]
//	Trim:   [1→100] [5→500] [100→3000]                              ← plateau removed
//	                                     ^^^^^^^^^^^^^^^^^^^^^^^^
//	                                     output stopped increasing = distorted rate
func CleanSamples(s [][2]*big.Int) [][2]*big.Int {
	s = lo.Filter(s, func(v [2]*big.Int, _ int) bool {
		return v[0] != nil && v[1] != nil && v[1].Sign() > 0
	})
	sort.Slice(s, func(a, b int) bool { return s[a][0].Cmp(s[b][0]) < 0 })
	s = lo.UniqBy(s, func(v [2]*big.Int) string { return v[0].String() })

	if len(s) >= 2 {
		cut := len(s)
		for i := len(s) - 1; i > 0; i-- {
			if s[i][1].Cmp(s[i-1][1]) <= 0 {
				cut = i
			} else {
				break
			}
		}
		s = s[:cut]
	}
	return s
}

// ValidRangeFromSamples: extract [min, max] amountIn from last run's cleaned samples.
//
//	samples[dir] = [[100→X], [500→Y], [3000→Z]]  →  prevMin=100, prevMax=3000
//	samples[dir] = []                              →  nil, nil (triggers cold start)
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

// FindCapBoundary: locate the out>0 → out==0 transition 
//
//	[1→100] [10→500] [100→3000] [1000→0] [10000→0]
//	                   ^^^^^^^^   ^^^^^^^
//	                   capLower   capUpper
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

// RefineCapPoints: evenly spaced between capLower and capUpper to narrow the real cap.
//
//	capLower=100, capUpper=1000 → [250, 400, 550, 700, 850]
//	                                ↑ query these to find exact transition
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

// ApplyBuffer: conservative safety margin — scale down all amountOut.
//
//	buffer=9970 (BasisPoint=10000) → keep 99.7% of each amountOut
//	[100→3000] becomes [100→2991]
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
