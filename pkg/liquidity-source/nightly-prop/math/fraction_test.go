package math

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func fraction(t testing.TB, raw *uint256.Int) Fraction {
	t.Helper()
	f, err := FractionFromRaw(raw)
	require.NoError(t, err)
	return f
}

func TestFractionFromRaw(t *testing.T) {
	require.True(t, fraction(t, new(uint256.Int)).IsLower())
	require.True(t, fraction(t, maxRaw).IsUpper())
	require.Equal(t, maxRaw, Upper.Raw())

	_, err := FractionFromRaw(new(uint256.Int).AddUint64(maxRaw, 1))
	require.ErrorIs(t, err, ErrOverflow)

	// Raw returns a copy, so a caller cannot change the fraction.
	f := fraction(t, u(5))
	f.Raw().SetUint64(6)
	require.Equal(t, u(5), f.Raw())
}

func TestFraction_MoveUp(t *testing.T) {
	nearUpper := fraction(t, new(uint256.Int).SubUint64(maxRaw, 1))
	partial, err := nearUpper.MoveUp(u(1), u(2), Floor)
	require.NoError(t, err)
	require.Equal(t, nearUpper, partial)
	drained, err := nearUpper.MoveUp(u(2), u(2), Floor)
	require.NoError(t, err)
	require.True(t, drained.IsUpper())
}

func TestFraction_MoveDown(t *testing.T) {
	nearLower := fraction(t, u(1))
	floor, err := nearLower.MoveDown(u(1), u(2), Floor)
	require.NoError(t, err)
	require.Equal(t, u(1), floor.Raw())
	ceil, err := nearLower.MoveDown(u(1), u(2), Ceil)
	require.NoError(t, err)
	require.True(t, ceil.IsLower())
	drained, err := nearLower.MoveDown(u(2), u(2), Floor)
	require.NoError(t, err)
	require.True(t, drained.IsLower())
}

// TestFraction_MoveError: the caller must supply amount <= depth and a nonzero depth, and the
// contract reverts otherwise.
func TestFraction_MoveError(t *testing.T) {
	// The move passes the upper edge, and SafeCast.toUint104 reverts.
	_, err := Lower.MoveUp(u(2), u(1), Floor)
	require.ErrorIs(t, err, ErrOverflow)
	// The move passes the lower edge, and the checked subtraction reverts.
	_, err = Upper.MoveDown(u(2), u(1), Floor)
	require.ErrorIs(t, err, ErrOverflow)
	_, err = Lower.MoveUp(u(0), u(0), Floor)
	require.ErrorIs(t, err, ErrDivisionByZero)
}

// randomMove returns a random fraction, a nonzero depth, and an amount up to the depth, as the
// walks supply.
func randomMove(t *testing.T, r *rand.Rand) (Fraction, *uint256.Int, *uint256.Int) {
	f := fraction(t, randomBits(r, 104))
	depth := randomBits(r, 256)
	if depth.IsZero() {
		depth = u(1)
	}
	var amount *uint256.Int
	switch r.IntN(10) {
	case 0:
		amount = new(uint256.Int)
	case 1:
		amount = new(uint256.Int).Set(depth)
	default:
		capped := Min(randomBits(r, depth.BitLen()), depth)
		amount = &capped
	}
	return f, depth, amount
}

// TestFraction_MoveUpMatchesExact: the floor move is the exact move (maxRaw - f) * amount / depth
// rounded down, the ceiling move is at most one more, and the whole depth reaches the upper edge.
func TestFraction_MoveUpMatchesExact(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 4))
	for range 20_000 {
		f, depth, amount := randomMove(t, r)
		floor, err := f.MoveUp(amount, depth, Floor)
		require.NoError(t, err)
		ceil, err := f.MoveUp(amount, depth, Ceil)
		require.NoError(t, err)
		distance := new(big.Int).Sub(maxRaw.ToBig(), f.Raw().ToBig())
		exact := distance.Mul(distance, amount.ToBig())
		exact.Quo(exact, depth.ToBig())
		msg := f.Raw().Dec() + " up by " + amount.Dec() + " / " + depth.Dec()
		require.Equal(t, exact.String(), new(big.Int).Sub(floor.Raw().ToBig(), f.Raw().ToBig()).String(), msg)
		require.LessOrEqual(t, new(uint256.Int).Sub(ceil.Raw(), floor.Raw()).Uint64(), uint64(1), msg)
		if amount.Eq(depth) {
			require.True(t, floor.IsUpper(), msg)
		}
	}
}

// TestFraction_MoveDownMatchesExact: the floor move is the exact move f * amount / depth rounded
// down, the ceiling move is at most one more, and the whole depth reaches the lower edge.
func TestFraction_MoveDownMatchesExact(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 5))
	for range 20_000 {
		f, depth, amount := randomMove(t, r)
		floor, err := f.MoveDown(amount, depth, Floor)
		require.NoError(t, err)
		ceil, err := f.MoveDown(amount, depth, Ceil)
		require.NoError(t, err)
		exact := new(big.Int).Mul(f.Raw().ToBig(), amount.ToBig())
		exact.Quo(exact, depth.ToBig())
		msg := f.Raw().Dec() + " down by " + amount.Dec() + " / " + depth.Dec()
		require.Equal(t, exact.String(), new(big.Int).Sub(f.Raw().ToBig(), floor.Raw().ToBig()).String(), msg)
		require.LessOrEqual(t, new(uint256.Int).Sub(floor.Raw(), ceil.Raw()).Uint64(), uint64(1), msg)
		if amount.Eq(depth) {
			require.True(t, floor.IsLower(), msg)
		}
	}
}
