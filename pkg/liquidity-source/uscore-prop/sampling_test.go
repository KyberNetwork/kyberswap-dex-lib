package uscoreprop

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func testAmounts(values ...uint64) []*uint256.Int {
	result := make([]*uint256.Int, len(values))
	for i, value := range values {
		result[i] = uint256.NewInt(value)
	}
	return result
}

func ladders(t *testing.T, p entity.Pool) [2][]Point {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	return extra.Ladders
}

func TestUpperRefinement(t *testing.T) {
	points := testAmounts(10, 625, 990)
	for range 2 {
		result := ladderQuote{status: 3, outs: make([]*uint256.Int, len(points))}
		for i, x := range points {
			result.outs[i] = new(uint256.Int)
			if x.Uint64() < 950 {
				result.outs[i].Set(x)
			}
		}
		points = refineUpper(points, result)
		require.LessOrEqual(t, len(points), maxSamplePoints)
	}
	last := uint64(0)
	for i, x := range points {
		if i > 0 {
			require.Positive(t, x.Cmp(points[i-1]))
		}
		if x.Uint64() < 950 {
			last = x.Uint64()
		}
	}
	require.GreaterOrEqual(t, last, uint64(944))
	for _, status := range []uint8{0, 1, 2, 4, 5} {
		require.Nil(t, refineUpper(testAmounts(1, 2), ladderQuote{status: status, outs: testAmounts(1, 0)}))
	}
	require.Len(t, refineUpper(testAmounts(1, 2), ladderQuote{status: 3, outs: testAmounts(1, 0)}), 2)
	require.Nil(t, refineUpper(testAmounts(1, 2), ladderQuote{status: 3, outs: testAmounts(0, 0)}))
}

func TestCurveRefinementFindsPeak(t *testing.T) {
	points := testAmounts(100, 200, 300)
	refined := refineCurve(points, ladderQuote{outs: testAmounts(80, 110, 105)})
	require.Contains(t, refined, uint256.NewInt(250))
	outs := make([]*uint256.Int, len(refined))
	for i, x := range refined {
		v := x.Uint64()
		switch {
		case v <= 100:
			outs[i] = uint256.NewInt(v * 8 / 10)
		case v <= 200:
			outs[i] = uint256.NewInt(80 + (v-100)*3/10)
		case v <= 250:
			outs[i] = uint256.NewInt(110 + (v-200)/5)
		default:
			outs[i] = uint256.NewInt(120 - (v-250)*3/10)
		}
	}
	quotes := collectQuotes(refined, ladderQuote{outs: outs})
	require.Equal(t, uint64(120), quotes[len(quotes)-1][1].Uint64())
	require.GreaterOrEqual(t, quotes[len(quotes)-1][0].Uint64(), uint64(250))
	require.LessOrEqual(t, quotes[len(quotes)-1][0].Uint64(), uint64(253))
	flat := collectQuotes(testAmounts(1, 2, 3), ladderQuote{outs: testAmounts(1, 1, 2)})
	require.Len(t, flat, 3)
}

func TestCurveRefinementScaleAndNeighbours(t *testing.T) {
	for _, exponent := range []int{0, 6, 18, 30} {
		scale := big256.TenPow(exponent)
		points, outs := testAmounts(100, 200, 300, 400), testAmounts(200, 400, 590, 750)
		for i := range points {
			points[i].Mul(points[i], scale)
			outs[i].Mul(outs[i], scale)
		}
		refined := refineCurve(points, ladderQuote{outs: outs})
		require.LessOrEqual(t, len(refined), maxSamplePoints)
		require.Equal(t, points[0], refined[0])
		require.Contains(t, refined, new(uint256.Int).Mul(uint256.NewInt(150), scale))
		require.Contains(t, refined, new(uint256.Int).Mul(uint256.NewInt(250), scale))
		for i := 1; i < len(refined); i++ {
			require.Positive(t, refined[i].Cmp(refined[i-1]))
		}
	}
	points := testAmounts(10, 100, 1000)
	require.Equal(t, points, refineCurve(points, ladderQuote{outs: testAmounts(20, 200, 2000)}))
	for _, status := range []uint8{1, 2, 4, 5} {
		require.Nil(t, refineCurve(points, ladderQuote{status: status}))
	}
}

func TestCurveRefinementBudget(t *testing.T) {
	points, outs := make([]*uint256.Int, 90), make([]*uint256.Int, 90)
	for i := range points {
		points[i] = uint256.NewInt(uint64(i+1) * 100)
		outs[i] = uint256.NewInt(1000 + uint64(i)*100)
	}
	refined := refineCurve(points, ladderQuote{outs: outs})
	require.Greater(t, len(refined), len(points))
	require.LessOrEqual(t, len(refined), maxSamplePoints)
	for i := 1; i < len(refined); i++ {
		require.Positive(t, refined[i].Cmp(refined[i-1]))
	}
	require.Nil(t, refineUpper(refined, ladderQuote{status: 3}))
}

// Samples past the peak or the first unquotable amount never reach the ladder, so refinement must
// drop them and spend that budget on the curve (a depleted pool's knee needs it most).
func TestCurveRefinementDropsDeadTail(t *testing.T) {
	points := testAmounts(100, 200, 300, 400, 500, 600, 700, 800)
	for _, outs := range [][]uint64{{100, 190, 270, 300, 290, 0, 0, 0}, {100, 190, 270, 300, 0, 0, 0, 0}} {
		refined := refineCurve(points, ladderQuote{status: 3, outs: testAmounts(outs...)})
		require.Equal(t, uint64(500), refined[len(refined)-1].Uint64())
		require.LessOrEqual(t, len(refined), maxSamplePoints)
	}
}
