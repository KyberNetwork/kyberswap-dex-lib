package uscore

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestInterpolationPolynomials(t *testing.T) {
	for degree := 1; degree <= 3; degree++ {
		for _, scale := range []float64{1, 1e6, 1e18, 1e30} {
			var points []ladder.Point
			for _, x := range []float64{1, 2, 4, 8} {
				points = append(points, ladder.Point{x * scale, math.Pow(x, float64(degree)) * scale})
			}
			for i := 1; i <= 128; i++ {
				x := float64(i) / 16
				out, err := interpolate(points, x*scale)
				require.NoError(t, err)
				require.InEpsilon(t, math.Pow(x, float64(degree))*scale, out, 1e-10)
			}
		}
	}
}

func checkInterpolation(t *testing.T, points []ladder.Point) {
	t.Helper()
	x0, y0 := 0.0, 0.0
	for _, point := range points {
		x1, y1 := point.AmountIn(), point.AmountOut()
		previous := y0
		for step := 1; step <= 32; step++ {
			x := x0 + (x1-x0)*float64(step)/32
			x = min(x, x1)
			out, err := interpolate(points, x)
			require.NoError(t, err)
			require.True(t, finite(out))
			require.GreaterOrEqual(t, out, y0)
			require.LessOrEqual(t, out, y1)
			require.GreaterOrEqual(t, out+math.Abs(y1)*1e-14, previous)
			previous = out
		}
		out, err := interpolate(points, x1)
		require.NoError(t, err)
		require.Equal(t, y1, out)
		x0, y0 = x1, y1
	}
}

func TestInterpolationBounds(t *testing.T) {
	for _, points := range [][]ladder.Point{
		{{1, 2}},
		{{1, 2}, {2, 2}, {4, 3}, {8, 3}},
		{{1, 1}, {2, 1000}, {3, 1001}, {10000, 1002}},
		{{1e-200, 1e-200}, {1e-100, 1e100}, {1, 1e200}, {1e100, 1e300}},
		{{1, 1}, {100, 1}, {1e20, 2}, {1e21, 2}},
	} {
		checkInterpolation(t, points)
	}
	_, err := interpolate(nil, 1)
	require.ErrorIs(t, err, ladder.ErrNoQuote)
	for _, amount := range []float64{0, -1, math.NaN(), math.Inf(-1)} {
		_, err = interpolate([]ladder.Point{{1, 1}}, amount)
		require.ErrorIs(t, err, ladder.ErrZeroAmountIn)
	}
	for _, amount := range []float64{2, math.Inf(1)} {
		_, err = interpolate([]ladder.Point{{1, 1}}, amount)
		require.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	}
}

func FuzzInterpolationMonotone(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	f.Add([]byte{255, 0, 0, 255, 1, 0, 255, 1, 0, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		sx, sy := math.Pow10(int(data[0]%61)-30), math.Pow10(int(data[1]%61)-30)
		x, y := 0.0, 0.0
		points := make([]ladder.Point, 0, min(len(data)/2, 32))
		for i := 2; i+1 < len(data) && len(points) < 32; i += 2 {
			x += float64(1+uint32(data[i])*uint32(data[i])) * sx
			y += float64(data[i+1]) * sy
			points = append(points, ladder.Point{x, y})
		}
		checkInterpolation(t, points)
	})
}

func TestSimulatorRejectsMalformedCurve(t *testing.T) {
	for _, points := range [][]ladder.Point{
		{{0, 1}}, {{-1, 1}}, {{1, 0}}, {{1, -1}},
		{{2, 1}, {1, 2}}, {{1, 1}, {1, 2}}, {{1, 2}, {2, 1}},
	} {
		p := testEntity()
		extra, err := json.Marshal(ladder.Extra{Ladders: [2][]ladder.Point{points, nil}})
		require.NoError(t, err)
		p.Extra = string(extra)
		_, err = NewPoolSimulatorWith(p, math.MaxInt64)
		require.ErrorIs(t, err, ErrInvalidState)
	}
}

func TestSimulatorSplitAndClone(t *testing.T) {
	p := testEntity()
	p.Reserves = []string{"1000000", "1000000"}
	p.Extra = `{"l":[[[1000,1900],[2000,3700],[4000,7000],[8000,13000]],[[1000,450],[2000,880],[4000,1600],[8000,2800]]]}`
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
		delta := new(big.Int).Sub(whole.TokenAmountOut.Amount, sum)
		require.LessOrEqual(t, delta.Abs(delta).Int64(), int64(1))
		require.Equal(t, [2]float64{}, original.consumedIn)
		require.Equal(t, [2]float64{}, original.consumedOut)
		require.Equal(t, "1000000", original.GetReserves()[dir].String())
		require.Equal(t, "1003000", sim.GetReserves()[dir].String())
	}
}

func TestSimulatorInvalidAmountAndUpdate(t *testing.T) {
	p := testEntity()
	p.Reserves = []string{"1000", "1000"}
	p.Extra = `{"l":[[[100,200],[200,390]],[[200,100],[400,195]]]}`
	sim, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1)} {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: amount}, TokenOut: testBase})
		require.ErrorIs(t, err, ladder.ErrZeroAmountIn)
	}
	for _, amounts := range [][2]*big.Int{
		{nil, nil}, {big.NewInt(-1), big.NewInt(1)}, {big.NewInt(1), big.NewInt(-1)},
		{big.NewInt(1), big.NewInt(1001)}, {new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)},
		{new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), big.NewInt(1)},
	} {
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: amounts[0]}, TokenAmountOut: pool.TokenAmount{Token: testBase, Amount: amounts[1]}})
		require.Equal(t, [2]float64{}, sim.consumedIn)
		require.Equal(t, [2]float64{}, sim.consumedOut)
		require.Equal(t, "1000", sim.GetReserves()[0].String())
		require.Equal(t, "1000", sim.GetReserves()[1].String())
	}
}
