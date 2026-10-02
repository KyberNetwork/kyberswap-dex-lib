package uscore

import (
	"math"
	"sort"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

func interpolate(points []ladder.Point, amount float64) (float64, error) {
	if amount <= 0 || math.IsNaN(amount) {
		return 0, ladder.ErrZeroAmountIn
	}
	if len(points) == 0 {
		return 0, ladder.ErrNoQuote
	}
	i := sort.Search(len(points), func(i int) bool { return points[i].AmountIn() >= amount })
	if i == len(points) {
		return 0, ladder.ErrAmountInTooLarge
	}
	if points[i].AmountIn() == amount {
		return points[i].AmountOut(), nil
	}
	x0, y0 := 0.0, 0.0
	if i > 0 {
		x0, y0 = points[i-1].AmountIn(), points[i-1].AmountOut()
	}
	x1, y1 := points[i].AmountIn(), points[i].AmountOut()
	h, dy := x1-x0, y1-y0
	if dy == 0 {
		return y0, nil
	}
	n := min(4, len(points)+1)
	start := max(0, min(i-1, len(points)+1-n))
	var nodes, coeff [4]float64
	for k := range n {
		x, y := 0.0, 0.0
		if start+k > 0 {
			x, y = points[start+k-1].AmountIn(), points[start+k-1].AmountOut()
		}
		nodes[k] = (x - x0) / h
		coeff[k] = (y - y0) / dy
	}
	for order := 1; order < n; order++ {
		for k := n - 1; k >= order; k-- {
			coeff[k] = (coeff[k] - coeff[k-1]) / (nodes[k] - nodes[k-order])
		}
	}
	slope := func(z float64) float64 {
		value, derivative := coeff[n-1], 0.0
		for k := n - 2; k >= 0; k-- {
			derivative = derivative*(z-nodes[k]) + value
			value = value*(z-nodes[k]) + coeff[k]
		}
		return derivative
	}
	left, right := slope(0), slope(1)
	if !finite(left) || !finite(right) {
		left, right = 1, 1
	}
	left, right = max(0, min(3, left)), max(0, min(3, right))
	u := (amount - x0) / h
	fraction := u * (left + u*(3-(2*left+right)+u*(left+right-2)))
	out := y0 + max(0, min(1, fraction))*dy
	return math.Max(y0, math.Min(y1, out)), nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
