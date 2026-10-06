package uscoreprop

import (
	"math"
	"math/big"
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// interpolateFloat is the float64 interpolation this package used before the fixed-point port,
// kept as the reference the port must match.
func interpolateFloat(points [][2]float64, amount float64) float64 {
	i := sort.Search(len(points), func(i int) bool { return points[i][0] >= amount })
	if points[i][0] == amount {
		return points[i][1]
	}
	x0, y0 := 0.0, 0.0
	if i > 0 {
		x0, y0 = points[i-1][0], points[i-1][1]
	}
	x1, y1 := points[i][0], points[i][1]
	h, dy := x1-x0, y1-y0
	if dy == 0 {
		return y0
	}
	n := min(4, len(points)+1)
	start := max(0, min(i-1, len(points)+1-n))
	var nodes, coeff [4]float64
	for k := range n {
		x, y := 0.0, 0.0
		if start+k > 0 {
			x, y = points[start+k-1][0], points[start+k-1][1]
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
	left, right = max(0, min(3, left)), max(0, min(3, right))
	u := (amount - x0) / h
	fraction := u * (left + u*(3-(2*left+right)+u*(left+right-2)))
	return y0 + max(0, min(1, fraction))*dy
}

func curve(xy ...*uint256.Int) []Point {
	points := make([]Point, len(xy)/2)
	for i := range points {
		points[i] = Point{xy[2*i], xy[2*i+1]}
	}
	return points
}

func at(t *testing.T, points []Point, amount *uint256.Int) *uint256.Int {
	t.Helper()
	var out uint256.Int
	require.NoError(t, interpolate(points, segmentSlopes(points), amount, &out))
	return &out
}

func requireClose(t *testing.T, want, got *uint256.Int, rel float64, msg ...any) {
	t.Helper()
	diff := new(big.Int).Sub(got.ToBig(), want.ToBig())
	tolerance, _ := new(big.Float).Mul(new(big.Float).SetInt(want.ToBig()), big.NewFloat(rel)).Int(nil)
	require.LessOrEqual(t, diff.Abs(diff).Cmp(tolerance.Add(tolerance, big.NewInt(2))), 0, msg...)
}

func TestInterpolationPolynomials(t *testing.T) {
	for degree := uint64(1); degree <= 3; degree++ {
		for _, scale := range []*uint256.Int{uint256.NewInt(1e6), uint256.NewInt(1e18), uint256.MustFromDecimal("1000000000000000000000000000000")} {
			var xy []*uint256.Int
			for _, x := range []uint64{1, 2, 4, 8} {
				xy = append(xy, new(uint256.Int).Mul(uint256.NewInt(x), scale), new(uint256.Int).Mul(uint256.NewInt(pow(x, degree)), scale))
			}
			points := curve(xy...)
			for i := uint64(1); i <= 128; i++ {
				amount := new(uint256.Int).Mul(uint256.NewInt(i), scale)
				amount.Rsh(amount, 4)
				want := new(uint256.Int).Mul(uint256.NewInt(pow(i, degree)), scale)
				want.Rsh(want, uint(4*degree))
				requireClose(t, want, at(t, points, amount), 1e-15, "degree=%d scale=%s i=%d", degree, scale, i)
			}
		}
	}
}

func pow(x, n uint64) uint64 {
	r := uint64(1)
	for range n {
		r *= x
	}
	return r
}

// A linear curve must be exact to the wei at any size; float64 drifts by ~1e8 wei at 1e24.
func TestInterpolationLinearIsExact(t *testing.T) {
	e24 := uint256.MustFromDecimal("1000000000000000000000000")
	var xy []*uint256.Int
	for k := uint64(1); k <= 6; k++ {
		x := new(uint256.Int).Mul(e24, uint256.NewInt(k))
		x.AddUint64(x, 123_456_789*k)
		xy = append(xy, x, new(uint256.Int).Mul(x, uint256.NewInt(3)))
	}
	points := curve(xy...)
	rng := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		amount := new(uint256.Int).SetUint64(rng.Uint64())
		amount.Mul(amount, uint256.NewInt(rng.Uint64N(300_000)+1)).AddUint64(amount, 1)
		if amount.Gt(points[len(points)-1][0]) {
			continue
		}
		require.Equal(t, new(uint256.Int).Mul(amount, uint256.NewInt(3)), at(t, points, amount), "amount=%s", amount)
	}
}

// The fixed-point port must reproduce the float64 implementation it replaced on well-conditioned
// curves (geometric samples of an oracle rate with a size-dependent fee, plus a depletion tail).
func TestInterpolationMatchesFloatReference(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	worst := 0.0
	for range 200 {
		n := 4 + rng.IntN(60)
		unit := math.Pow10(6 + rng.IntN(19))
		rate := math.Pow10(rng.IntN(25)-12) * (1 + rng.Float64())
		depletion := math.Inf(1)
		if rng.IntN(2) == 0 {
			depletion = unit * math.Pow(1.25, float64(n)) * rate * (0.3 + 0.5*rng.Float64())
		}
		var points []Point
		var ref [][2]float64
		x := unit * (0.001 + rng.Float64()*0.01)
		for range n {
			x *= 1.05 + rng.Float64()*0.4
			y := math.Min(depletion, x*rate*(1-0.0005-0.02*x/(x+unit*1e4)))
			xi, _ := new(big.Float).SetFloat64(x).Int(nil)
			yi, _ := new(big.Float).SetFloat64(math.Floor(y)).Int(nil)
			if yi.Sign() <= 0 || len(points) > 0 && yi.Cmp(points[len(points)-1][1].ToBig()) < 0 {
				continue
			}
			p := Point{uint256.MustFromBig(xi), uint256.MustFromBig(yi)}
			points = append(points, p)
			ref = append(ref, [2]float64{p[0].Float64(), p[1].Float64()})
		}
		if len(points) == 0 {
			continue
		}
		slopes := segmentSlopes(points)
		for range 50 {
			last := points[len(points)-1][0].Float64()
			amount, _ := new(big.Float).SetFloat64(last * rng.Float64()).Int(nil)
			if amount.Sign() <= 0 {
				continue
			}
			var got uint256.Int
			require.NoError(t, interpolate(points, slopes, uint256.MustFromBig(amount), &got))
			af, _ := amount.Float64()
			wantF, _ := new(big.Float).SetFloat64(interpolateFloat(ref, af)).Int(nil)
			want := uint256.MustFromBig(wantF)
			requireClose(t, want, &got, 1e-12, "amount=%s", amount)
			if !want.IsZero() {
				d := new(big.Int).Sub(got.ToBig(), want.ToBig())
				rel, _ := new(big.Rat).SetFrac(d.Abs(d), want.ToBig()).Float64()
				worst = max(worst, rel)
			}
		}
	}
	t.Logf("max relative difference vs float64 reference: %.3g", worst)
}

func checkInterpolation(t *testing.T, points []Point) {
	t.Helper()
	slopes := segmentSlopes(points)
	x0, y0 := uint256.NewInt(0), uint256.NewInt(0)
	var out, previous, x, step uint256.Int
	for _, p := range points {
		x1, y1 := p[0], p[1]
		previous.Set(y0)
		step.Sub(x1, x0)
		for k := uint64(1); k <= 32; k++ {
			x.MulDivOverflow(&step, uint256.NewInt(k), uint256.NewInt(32))
			if x.Add(&x, x0).IsZero() {
				continue
			}
			require.NoError(t, interpolate(points, slopes, &x, &out))
			require.False(t, out.Lt(y0), "below segment start")
			require.False(t, out.Gt(y1), "above segment end")
			require.False(t, out.Lt(&previous), "not monotone")
			previous.Set(&out)
		}
		require.NoError(t, interpolate(points, slopes, x1, &out))
		require.Equal(t, y1, &out)
		x0, y0 = x1, y1
	}
}

func TestInterpolationBounds(t *testing.T) {
	n := uint256.NewInt
	for _, points := range [][]Point{
		curve(n(1), n(2)),
		curve(n(1), n(2), n(2), n(2), n(4), n(3), n(8), n(3)),
		curve(n(1), n(1), n(2), n(1000), n(3), n(1001), n(10000), n(1002)),
		curve(n(1), n(1), n(100), n(1), uint256.MustFromDecimal("100000000000000000000"), n(2), uint256.MustFromDecimal("1000000000000000000000"), n(2)),
		curve(n(1), n(1), n(2), maxPoint),
		curve(n(1), n(1), new(uint256.Int).Rsh(maxPoint, 1), n(2), maxPoint, maxPoint),
	} {
		checkInterpolation(t, points)
	}
	var out uint256.Int
	require.ErrorIs(t, interpolate(nil, nil, n(1), &out), ladder.ErrNoQuote)
	one := curve(n(1), n(1))
	require.ErrorIs(t, interpolate(one, segmentSlopes(one), n(0), &out), ladder.ErrZeroAmountIn)
	require.ErrorIs(t, interpolate(one, segmentSlopes(one), n(2), &out), ladder.ErrAmountInTooLarge)
}

func FuzzInterpolationMonotone(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	f.Add([]byte{255, 0, 0, 255, 1, 0, 255, 1, 0, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		sx, sy := new(uint256.Int).Lsh(uint256.NewInt(1), uint(data[0]%100)), new(uint256.Int).Lsh(uint256.NewInt(1), uint(data[1]%100))
		x, y := new(uint256.Int), new(uint256.Int)
		var points []Point
		for i := 2; i+1 < len(data) && len(points) < 32; i += 2 {
			x = new(uint256.Int).Add(x, new(uint256.Int).Mul(uint256.NewInt(1+uint64(data[i])*uint64(data[i])), sx))
			y = new(uint256.Int).Add(y, new(uint256.Int).Mul(uint256.NewInt(uint64(data[i+1])), sy))
			if y.IsZero() {
				continue
			}
			points = append(points, Point{x, y})
		}
		checkInterpolation(t, points)
	})
}

func TestSimulatorRejectsMalformedCurve(t *testing.T) {
	for _, ladder := range []string{
		`[["0","1"]]`, `[["1","0"]]`, `[[null,"1"]]`, `[["1"]]`,
		`[["2","1"],["1","2"]]`, `[["1","1"],["1","2"]]`, `[["1","2"],["2","1"]]`,
		`[["340282366920938463463374607431768211456","1"]]`, `[["1","340282366920938463463374607431768211456"]]`,
	} {
		p := testEntity()
		p.Extra = `{"l":[` + ladder + `,null]}`
		_, err := NewPoolSimulatorWith(p, math.MaxInt64)
		require.ErrorIs(t, err, ErrInvalidState, ladder)
	}
}

func TestSimulatorSplitAndClone(t *testing.T) {
	p := testEntity()
	p.Reserves = []string{"1000000", "1000000"}
	p.Extra = `{"l":[[["1000","1900"],["2000","3700"],["4000","7000"],["8000","13000"]],[["1000","450"],["2000","880"],["4000","1600"],["8000","2800"]]]}`
	original, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	for dir := range 2 {
		sim := original.CloneState().(*PoolSimulator)
		q := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[dir].Address, Amount: big.NewInt(3000)}, TokenOut: p.Tokens[1-dir].Address}
		whole, err := sim.CalcAmountOut(q)
		require.NoError(t, err)
		sum := new(big.Int)
		q.TokenAmountIn.Amount = big.NewInt(100)
		for range 30 {
			quote, err := sim.CalcAmountOut(q)
			require.NoError(t, err)
			again, err := sim.CalcAmountOut(q)
			require.NoError(t, err)
			require.Equal(t, quote, again)
			sum.Add(sum, quote.TokenAmountOut.Amount)
			sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *quote.TokenAmountOut, Fee: *quote.Fee})
		}
		require.Equal(t, whole.TokenAmountOut.Amount, sum, "split route must walk the same curve")
		require.Equal(t, [2]uint256.Int{}, original.consumedIn)
		require.Equal(t, [2]uint256.Int{}, original.consumedOut)
		require.Equal(t, "1000000", original.GetReserves()[dir].String())
		require.Equal(t, "1003000", sim.GetReserves()[dir].String())
	}
}

func TestSimulatorInvalidAmountAndUpdate(t *testing.T) {
	p := testEntity()
	p.Reserves = []string{"1000", "1000"}
	p.Extra = `{"l":[[["100","200"],["200","390"]],[["200","100"],["400","195"]]]}`
	sim, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1)} {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: amount}, TokenOut: testBase})
		require.ErrorIs(t, err, ladder.ErrZeroAmountIn)
	}
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: new(big.Int).Lsh(big.NewInt(1), 256)}, TokenOut: testBase})
	require.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	for _, amounts := range [][2]*big.Int{
		{nil, nil}, {big.NewInt(-1), big.NewInt(1)}, {big.NewInt(1), big.NewInt(-1)},
		{big.NewInt(1), big.NewInt(1001)}, {new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)},
		{new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), big.NewInt(1)},
	} {
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: amounts[0]}, TokenAmountOut: pool.TokenAmount{Token: testBase, Amount: amounts[1]}})
		require.Equal(t, [2]uint256.Int{}, sim.consumedIn)
		require.Equal(t, [2]uint256.Int{}, sim.consumedOut)
		require.Equal(t, "1000", sim.GetReserves()[0].String())
		require.Equal(t, "1000", sim.GetReserves()[1].String())
	}
}
