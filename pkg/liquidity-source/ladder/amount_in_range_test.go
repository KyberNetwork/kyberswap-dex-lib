package ladder

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ pool.IAmountInRange = (*PoolSimulator)(nil)

// newLadderPool is newTestPool over a readable ladder literal: entity.Pool.Extra is a string
// field, so the literal has to be escaped into a JSON string before it gets there.
func newLadderPool(t *testing.T, ladders string) *PoolSimulator {
	t.Helper()
	quoted, err := json.Marshal(ladders)
	require.NoError(t, err)
	return newTestPool(t, string(quoted))
}

// A ladder quoted out to 100e18 one way and 100e6 the other. Reserves (set by newTestPool) are far
// above both outputs, so nothing but the extent binds.
const rangeExtra = `{"l":[
	[[1e18,1e6],[10e18,9.9e6],[100e18,98e6]],
	[[1e6,0.99e18],[10e6,9.8e18],[100e6,97e18]]
]}`

func TestPoolSimulator_AmountInRange(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, rangeExtra)

	t.Run("reports the ladder extent per direction", func(t *testing.T) {
		minIn, maxIn, err := s.AmountInRange(testToken0, testToken1, nil)
		require.NoError(t, err)
		assert.Nil(t, minIn, "the spline's knots start at zero, so there is no minimum")
		assert.Equal(t, "100000000000000000000", maxIn.String())

		minIn, maxIn, err = s.AmountInRange(testToken1, testToken0, nil)
		require.NoError(t, err)
		assert.Nil(t, minIn)
		assert.Equal(t, "100000000", maxIn.String(), "the other direction has its own extent")
	})

	t.Run("rejects a direction it does not hold", func(t *testing.T) {
		for _, tc := range [][2]string{
			{"0xnope", testToken1},
			{testToken0, "0xnope"},
			{testToken0, testToken0},
		} {
			_, _, err := s.AmountInRange(tc[0], tc[1], nil)
			assert.ErrorIs(t, err, ErrInvalidToken, "%v", tc)
		}
	})
}

// The bound is only worth anything if it is where the refusal actually is. This is the pin: quote
// at maxIn and above it, and require the pool to agree with what the range claimed.
func TestPoolSimulator_AmountInRangeMatchesWhereCalcAmountOutRefuses(t *testing.T) {
	t.Parallel()

	for _, dir := range [][2]string{{testToken0, testToken1}, {testToken1, testToken0}} {
		s := newLadderPool(t, rangeExtra)
		_, maxIn, err := s.AmountInRange(dir[0], dir[1], nil)
		require.NoError(t, err)

		_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: dir[0], Amount: maxIn},
			TokenOut:      dir[1],
		})
		assert.NoError(t, err, "maxIn itself must quote")

		// 1% above, not one wei above: the knots hold wei as float64, so near 1e20 a single wei is
		// below the representable spacing and CalcAmountOut would see the same amount back.
		above := new(big.Int).Div(new(big.Int).Mul(maxIn, big.NewInt(101)), big.NewInt(100))
		_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: dir[0], Amount: above},
			TokenOut:      dir[1],
		})
		assert.ErrorIs(t, err, ErrAmountInTooLarge, "above maxIn must be refused")
	}
}

// consumedIn is the half of this that a caller cannot see. CalcAmountOut quotes at
// consumedIn+amountIn, so a partly filled pool has less room than its ladder suggests, and a range
// that ignored that would overstate by everything already taken.
func TestPoolSimulator_AmountInRangeShrinksAsThePoolFills(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, rangeExtra)

	_, before, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)

	fill := bignumber.NewBig("30000000000000000000") // 30e18 of the 100e18 ladder
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenAmountOut: *res.TokenAmountOut,
	})

	_, after, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)
	assert.Equal(t, "70000000000000000000", after.String(), "100e18 ladder less the 30e18 filled")
	assert.Positive(t, before.Cmp(after))

	// And the other direction is untouched: consumedIn is per direction.
	_, other, err := s.AmountInRange(testToken1, testToken0, nil)
	require.NoError(t, err)
	assert.Equal(t, "100000000", other.String())
}

// A fully consumed ladder is a statement ("alive, quotes nothing"), not an absence of one, so it
// reports a zero maximum rather than claiming it cannot say.
func TestPoolSimulator_AmountInRangeZeroWhenExhausted(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, rangeExtra)

	fill := bignumber.NewBig("100000000000000000000") // the whole ladder
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenAmountOut: *res.TokenAmountOut,
	})

	minIn, maxIn, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err, "exhausted is reportable, not unsupported")
	assert.Nil(t, minIn)
	require.NotNil(t, maxIn)
	assert.Equal(t, 0, maxIn.Sign(), "a zero maximum spares the caller a doomed probe")
}

// A direction with no ladder -- which is how this package represents a paused pool or a side with
// no liquidity -- has no extent to report at all. ErrUnsupported sends the caller back to its own
// sampling instead of letting it read silence as a bound.
func TestPoolSimulator_AmountInRangeUnsupportedWithoutACurve(t *testing.T) {
	t.Parallel()

	for name, extra := range map[string]string{
		"both directions empty": `{"l":[[],[]]}`,
		"one direction empty":   `{"l":[[],[[1e6,0.99e18],[10e6,9.8e18]]]}`,
	} {
		t.Run(name, func(t *testing.T) {
			s := newLadderPool(t, extra)
			_, _, err := s.AmountInRange(testToken0, testToken1, nil)
			assert.ErrorIs(t, err, pool.ErrUnsupported)
		})
	}

	// The populated side of a one-sided pool still answers.
	s := newLadderPool(t, `{"l":[[],[[1e6,0.99e18],[10e6,9.8e18]]]}`)
	_, maxIn, err := s.AmountInRange(testToken1, testToken0, nil)
	require.NoError(t, err)
	assert.Equal(t, "10000000", maxIn.String())
}

// The extent has to come off the spline's knots, which are filtered and sorted, not off the raw
// Extra.Ladders. Here the snapshot's last point is out of order: NewSpline drops it, so the pool
// still quotes up to 100e18, and a range built from Extra.Ladders' last element would understate
// the pool by half -- the silent direction of error the interface forbids.
func TestPoolSimulator_AmountInRangeFollowsTheSplineNotTheRawSnapshot(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, `{"l":[
		[[1e18,1e6],[10e18,9.9e6],[100e18,98e6],[50e18,50e6]],
		[]
	]}`)

	require.Equal(t, 50e18, s.Extra.Ladders[0][len(s.Extra.Ladders[0])-1].AmountIn(),
		"fixture must keep the out-of-order point in the raw snapshot")

	_, maxIn, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)
	assert.Equal(t, "100000000000000000000", maxIn.String())

	// And the pool really does quote there, which is what makes 100e18 the honest answer.
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: maxIn},
		TokenOut:      testToken1,
	})
	assert.NoError(t, err)
}

// CloneState shares the splines by design, so a clone must still report its own fill level rather
// than the original's -- route assembly clones a pool per candidate split.
func TestPoolSimulator_AmountInRangeAfterCloneState(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, rangeExtra)

	fill := bignumber.NewBig("30000000000000000000")
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenOut:      testToken1,
	})
	require.NoError(t, err)

	clone := s.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken0, Amount: fill},
		TokenAmountOut: *res.TokenAmountOut,
	})

	_, cloned, err := clone.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)
	assert.Equal(t, "70000000000000000000", cloned.String())

	_, original, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)
	assert.Equal(t, "100000000000000000000", original.String(),
		"filling the clone must not shrink the original's range")
}

// A single knot is still a curve: NewSpline prepends the origin, so the spline interpolates (0,0)
// to that knot and the pool quotes anything up to it. The extent is the knot, not "cannot say" --
// worth pinning because one probed point looks degenerate and is not.
func TestPoolSimulator_AmountInRangeWithASingleKnot(t *testing.T) {
	t.Parallel()
	s := newLadderPool(t, `{"l":[[[1e18,1e6]],[]]}`)

	_, maxIn, err := s.AmountInRange(testToken0, testToken1, nil)
	require.NoError(t, err)
	assert.Equal(t, "1000000000000000000", maxIn.String())

	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: maxIn},
		TokenOut:      testToken1,
	})
	assert.NoError(t, err, "the knot quotes, so reporting it is honest")
}
