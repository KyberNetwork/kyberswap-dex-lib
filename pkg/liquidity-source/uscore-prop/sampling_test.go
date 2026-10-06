package uscoreprop

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func testAmounts(values ...int64) []*big.Int {
	result := make([]*big.Int, len(values))
	for i, value := range values {
		result[i] = big.NewInt(value)
	}
	return result
}

func TestTrackerSamplingIgnoresPreviousState(t *testing.T) {
	p := testEntity()
	tracker := NewPoolTracker(&Config{}, mockRPC(t, false, false, nil, nil))
	expected, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	for _, reserves := range [][]string{{"1000", "1000"}, {"1000000000000", "1000000000000"}, {"invalid", "invalid"}} {
		for _, extra := range []string{"", "{}", "{", `{"l":[[[100,200],[200,390],[300,410]],[[100,200],[200,390],[300,410]]]}`} {
			p.Reserves, p.Extra = reserves, extra
			actual, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, expected.Extra, actual.Extra)
			require.Equal(t, expected.Reserves, actual.Reserves)
		}
	}
}

func TestUpperRefinement(t *testing.T) {
	points := testAmounts(10, 625, 990)
	for range 2 {
		result := ladderResult{Status: 3, Outs: make([]*big.Int, len(points))}
		for i, x := range points {
			result.Outs[i] = big.NewInt(0)
			if x.Int64() < 950 {
				result.Outs[i].Set(x)
			}
		}
		points = refineUpper(points, result)
		require.LessOrEqual(t, len(points), maxSamplePoints)
	}
	last := int64(0)
	for i, x := range points {
		if i > 0 {
			require.Positive(t, x.Cmp(points[i-1]))
		}
		if x.Int64() < 950 {
			last = x.Int64()
		}
	}
	require.GreaterOrEqual(t, last, int64(944))
	for _, status := range []uint8{0, 1, 2, 4, 5} {
		require.Nil(t, refineUpper(testAmounts(1, 2), ladderResult{Status: status, Outs: testAmounts(1, 0)}))
	}
	require.Len(t, refineUpper(testAmounts(1, 2), ladderResult{Status: 3, Outs: testAmounts(1, 0)}), 2)
	require.Nil(t, refineUpper(testAmounts(1, 2), ladderResult{Status: 3, Outs: testAmounts(0, 0)}))
}

func TestCurveRefinementFindsPeak(t *testing.T) {
	points := testAmounts(100, 200, 300)
	refined := refineCurve(points, ladderResult{Outs: testAmounts(80, 110, 105)})
	require.Contains(t, refined, big.NewInt(250))
	outs := make([]*big.Int, len(refined))
	for i, x := range refined {
		v := x.Int64()
		switch {
		case v <= 100:
			outs[i] = big.NewInt(v * 8 / 10)
		case v <= 200:
			outs[i] = big.NewInt(80 + (v-100)*3/10)
		case v <= 250:
			outs[i] = big.NewInt(110 + (v-200)/5)
		default:
			outs[i] = big.NewInt(120 - (v-250)*3/10)
		}
	}
	quotes, err := collectQuotes(refined, ladderResult{Outs: outs, Fees: make([]*big.Int, len(refined))})
	require.NoError(t, err)
	require.Equal(t, float64(120), quotes[len(quotes)-1].AmountOut())
	require.GreaterOrEqual(t, quotes[len(quotes)-1].AmountIn(), float64(250))
	require.LessOrEqual(t, quotes[len(quotes)-1].AmountIn(), float64(253))
	flat, err := collectQuotes(testAmounts(1, 2, 3), ladderResult{Outs: testAmounts(1, 1, 2), Fees: testAmounts(0, 0, 0)})
	require.NoError(t, err)
	require.Len(t, flat, 3)
}

func TestCurveRefinementScaleAndNeighbours(t *testing.T) {
	for _, exponent := range []int64{0, 6, 18, 30} {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(exponent), nil)
		points, outs := testAmounts(100, 200, 300, 400), testAmounts(200, 400, 590, 750)
		for i := range points {
			points[i].Mul(points[i], scale)
			outs[i].Mul(outs[i], scale)
		}
		refined := refineCurve(points, ladderResult{Outs: outs})
		require.LessOrEqual(t, len(refined), maxSamplePoints)
		require.Equal(t, points[0], refined[0])
		require.Contains(t, refined, new(big.Int).Mul(big.NewInt(150), scale))
		require.Contains(t, refined, new(big.Int).Mul(big.NewInt(250), scale))
		for i := 1; i < len(refined); i++ {
			require.Positive(t, refined[i].Cmp(refined[i-1]))
		}
	}
	points := testAmounts(10, 100, 1000)
	require.Equal(t, points, refineCurve(points, ladderResult{Outs: testAmounts(20, 200, 2000)}))
	for _, status := range []uint8{1, 2, 4, 5} {
		require.Nil(t, refineCurve(points, ladderResult{Status: status}))
	}
}

func TestCurveRefinementBudget(t *testing.T) {
	points, outs := make([]*big.Int, 90), make([]*big.Int, 90)
	for i := range points {
		points[i] = big.NewInt(int64(i+1) * 100)
		outs[i] = big.NewInt(1000 + int64(i)*100)
	}
	refined := refineCurve(points, ladderResult{Outs: outs})
	require.Greater(t, len(refined), len(points))
	require.LessOrEqual(t, len(refined), maxSamplePoints)
	for i := 1; i < len(refined); i++ {
		require.Positive(t, refined[i].Cmp(refined[i-1]))
	}
	require.Nil(t, refineUpper(refined, ladderResult{Status: 3}))
}

func TestTrackerRefinementDiscardsEarlierSnapshots(t *testing.T) {
	for _, status := range []uint8{1, 2, 4, 5, 6} {
		t.Run(big.NewInt(int64(status)).String(), func(t *testing.T) {
			count := 0
			client := mockRPCWithStatus(t, false, false, &count, nil, nil, func(snapshot int) uint8 {
				if snapshot >= 2 {
					return status
				}
				return 0
			})
			p, err := NewPoolTracker(&Config{}, client).GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
			require.Equal(t, 3, count)
			if status == 6 {
				require.ErrorIs(t, err, ErrInvalidState)
				return
			}
			require.NoError(t, err)
			require.Equal(t, uint64(102), p.BlockNumber)
			require.Equal(t, int64(1_800_000_002), p.Timestamp)
			var extra ladder.Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
			require.Empty(t, extra.Ladders[0])
			require.Empty(t, extra.Ladders[1])
		})
	}
}

func TestTrackerRepeatedRefreshRetainsUpperRange(t *testing.T) {
	count := 0
	tracker := NewPoolTracker(&Config{}, mockRPC(t, false, false, &count, nil))
	p := testEntity()
	for range 5 {
		before := count
		var err error
		p, err = tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.LessOrEqual(t, count-before, 2+maxRefinementRounds)
		var extra ladder.Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		for _, points := range extra.Ladders {
			require.LessOrEqual(t, len(points), maxSamplePoints)
			require.Equal(t, float64(300), points[len(points)-1].AmountIn())
		}
	}
}
