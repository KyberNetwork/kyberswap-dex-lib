package everlongflamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// quoteCells is one cell, and quoteSecond a second-fill charge, for a ladder whose charge a test does not read: a
// curve quotes only with both.
var quoteCells = []Cell{{Gas: 100_000}}

const quoteSecond = 50_000

func pts(xy ...uint64) []Point {
	out := make([]Point, 0, len(xy)/2)
	for k := 0; k+1 < len(xy); k += 2 {
		out = append(out, Point{uint256.NewInt(xy[k]), uint256.NewInt(xy[k+1])})
	}
	return out
}

func quoteOf(t *testing.T, c *curve, amount, haircut uint64) (uint64, uint64, bool, error) {
	t.Helper()
	var used, out uint256.Int
	capped, err := c.quote(uint256.NewInt(amount), haircut, &used, &out)
	return used.Uint64(), out.Uint64(), capped, err
}

func TestCurveIsExactAtEveryProbe(t *testing.T) {
	c, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970, 400, 3900)})
	require.NoError(t, err)
	for _, p := range [][2]uint64{{100, 990}, {200, 1970}, {400, 3900}} {
		used, out, capped, err := quoteOf(t, &c, p[0], 50)
		require.NoError(t, err)
		require.False(t, capped)
		require.Equal(t, p[0], used)
		require.Equal(t, p[1], out, "a probed size pays the probe, no haircut")
	}
}

func TestCurveInterpolatesConcaveSegmentsUnderTheChord(t *testing.T) {
	// slopes 9.9, 9.8, 9.65: concave, so every segment takes the chord less the haircut
	c, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970, 400, 3900)})
	require.NoError(t, err)
	require.Equal(t, []bool{false, true, true}, c.chord)

	_, out, _, err := quoteOf(t, &c, 300, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2935), out) // 1970 + 1930 * 100 / 200

	_, out, _, err = quoteOf(t, &c, 300, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(2935*9_990/10_000), out)

	_, _, _, err = quoteOf(t, &c, 99, 0)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "nothing under the first probe: the pool may refuse it")

	// never below the probe under it: just past a probe the haircut would take more than the chord adds
	_, out, _, err = quoteOf(t, &c, 201, 100)
	require.NoError(t, err)
	require.Equal(t, uint64(1970), out)
}

func TestCurveStepsWhereTheProbesAreNotConcave(t *testing.T) {
	// a fixed cost (out = 10x - 100): the origin chord would pay more than the pool, and so would segment 1's
	// chord if the kink sat inside it; the probes cannot tell, so both step
	c, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 900, 200, 1900, 400, 3900)})
	require.NoError(t, err)
	require.Equal(t, []bool{false, false, true}, c.chord)

	_, _, _, err = quoteOf(t, &c, 50, 0)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "nothing under the first probe")

	_, out, _, err := quoteOf(t, &c, 150, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(900), out, "the step: the probe below")

	_, out, _, err = quoteOf(t, &c, 300, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2900), out, "a segment between two concave neighbours takes the chord")

	// a branch that pays more past a size (slope 9 then 12 then 9): the steeper segment and the one before it step
	c, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 950, 200, 1850, 300, 3050, 400, 3950)})
	require.NoError(t, err)
	require.Equal(t, []bool{false, false, false, true}, c.chord)
	_, out, _, err = quoteOf(t, &c, 250, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1850), out)
}

func TestCurveCeilingAndRefusals(t *testing.T) {
	c, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970), Capped: true})
	require.NoError(t, err)
	used, out, capped, err := quoteOf(t, &c, 1_000, 0)
	require.NoError(t, err)
	require.True(t, capped)
	require.Equal(t, uint64(200), used, "a larger input fills at the ceiling")
	require.Equal(t, uint64(1970), out)

	c, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970)})
	require.NoError(t, err)
	_, _, _, err = quoteOf(t, &c, 201, 0)
	require.ErrorIs(t, err, ladder.ErrAmountInTooLarge, "past an uncapped ladder's top: no quote")

	c, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990)})
	require.NoError(t, err)
	_, _, _, err = quoteOf(t, &c, 99, 0)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "one probe alone quotes only itself")
	_, out, _, err = quoteOf(t, &c, 100, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(990), out)
}

func TestCurveRefusesMalformedLadders(t *testing.T) {
	for name, l := range map[string][]Point{
		"input not rising": pts(100, 990, 100, 1000),
		"output falling":   pts(100, 990, 200, 980),
		"zero output":      pts(100, 0),
		"too large":        {{new(uint256.Int).Lsh(uint256.NewInt(1), maxPointBits), uint256.NewInt(1)}},
		"missing amount":   {{nil, uint256.NewInt(1)}},
	} {
		_, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: l})
		require.ErrorIs(t, err, ErrInvalidState, name)
	}
}

func TestCurveQuotesAFlatSpanAtThePointBelow(t *testing.T) {
	// slopes 9.9, 9.8, 9.65: all chords but span 2, which the ladder marks flat
	c, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970, 400, 3900), Flat: 1 << 2})
	require.NoError(t, err)
	require.Equal(t, []bool{false, true, false}, c.chord)
	require.True(t, c.flat)
	_, out, _, err := quoteOf(t, &c, 300, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1970), out, "the point below")
	_, out, _, err = quoteOf(t, &c, 400, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(3900), out, "a point is exact")
	_, out, _, err = quoteOf(t, &c, 150, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1480), out, "the chord")
	_, _, _, err = quoteOf(t, &c, 99, 0)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "nothing under the first point")

	// bit 0 covers no span, and bits past the last point none either
	c, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 990, 200, 1970), Flat: 1 | 1<<5})
	require.NoError(t, err)
	require.Equal(t, []bool{false, true}, c.chord)
	require.False(t, c.flat)

	// a span only `steeper` steps is no flat span
	c, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(100, 900, 200, 1900, 400, 3900)})
	require.NoError(t, err)
	require.Equal(t, []bool{false, false, true}, c.chord)
	require.False(t, c.flat)
}

func TestCurveRefusesALadderPastItsFlatBits(t *testing.T) {
	xy := make([]uint64, 0, 2*(maxLadderPoints+1))
	for k := range uint64(maxLadderPoints + 1) {
		xy = append(xy, 100*(k+1), 99*(k+1))
	}
	_, err := newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(xy...)})
	require.ErrorIs(t, err, ErrInvalidState, "Flat holds one bit a span")
	_, err = newCurve(Ladder{Cells: quoteCells, Second: quoteSecond, Points: pts(xy[:2*maxLadderPoints]...)})
	require.NoError(t, err)
}
