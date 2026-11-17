package math

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var (
	maxUint256 = new(uint256.Int).SetAllOne()
	two128     = new(uint256.Int).Lsh(one, 128)
	// bigLimit is 2^256, the first value that does not fit uint256.
	bigLimit = new(big.Int).Lsh(big.NewInt(1), 256)
)

func u(v uint64) *uint256.Int {
	return uint256.NewInt(v)
}

// randomBits returns a random value below 2^bits.
func randomBits(r *rand.Rand, bits int) *uint256.Int {
	z := new(uint256.Int)
	for i := range z {
		z[i] = r.Uint64()
	}
	return z.Rsh(z, uint(256-bits))
}

// randomUint returns a random value of a random bit length, so that small and large values both
// appear.
func randomUint(r *rand.Rand) *uint256.Int {
	return randomBits(r, r.IntN(257))
}

// requireFits checks got and err against want: an ErrOverflow when want does not fit uint256, and
// want otherwise.
func requireFits(t *testing.T, want *big.Int, got *uint256.Int, err error, msg string) {
	t.Helper()
	if want.Cmp(bigLimit) >= 0 {
		require.ErrorIs(t, err, ErrOverflow, msg)
		return
	}
	require.NoError(t, err, msg)
	// Compare decimal strings: two equal big.Int values can differ in their internal layout.
	require.Equal(t, want.String(), got.Dec(), msg)
}

func TestAddSubMul(t *testing.T) {
	sum, err := Add(u(2), u(3))
	require.NoError(t, err)
	require.Equal(t, u(5), sum)
	_, err = Add(maxUint256, one)
	require.ErrorIs(t, err, ErrOverflow)

	difference, err := Sub(u(3), u(2))
	require.NoError(t, err)
	require.Equal(t, one, difference)
	_, err = Sub(u(2), u(3))
	require.ErrorIs(t, err, ErrOverflow)

	product, err := Mul(u(6), u(7))
	require.NoError(t, err)
	require.Equal(t, u(42), product)
	_, err = Mul(two128, two128)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestMin(t *testing.T) {
	require.Equal(t, u(2), Min(u(2), u(3)))
	require.Equal(t, u(2), Min(u(3), u(2)))
}

func TestMulDiv(t *testing.T) {
	// (2^256 - 1)^2 / (2^256 - 1) needs the full 512-bit product.
	got, err := MulDiv(maxUint256, maxUint256, maxUint256, Floor)
	require.NoError(t, err)
	require.Equal(t, maxUint256, got)

	got, err = MulDiv(maxUint256, u(3), u(4), Floor)
	require.NoError(t, err)
	want := new(uint256.Int).Div(maxUint256, u(4))
	want.Mul(want, u(3)).AddUint64(want, 2)
	require.Equal(t, want, got)

	for _, c := range []struct {
		x, y, denominator uint64
		rounding          Rounding
		want              uint64
	}{
		{10, 1, 3, Floor, 3},
		{10, 1, 3, Ceil, 4},
		{9, 1, 3, Ceil, 3},
		{0, 5, 3, Ceil, 0},
	} {
		got, err := MulDiv(u(c.x), u(c.y), u(c.denominator), c.rounding)
		require.NoError(t, err)
		require.Equal(t, u(c.want), got, "%d * %d / %d", c.x, c.y, c.denominator)
	}
}

// TestMulDiv_DivisionByZero: a zero denominator wins over a product that does not fit 256 bits, as
// in OpenZeppelin.
func TestMulDiv_DivisionByZero(t *testing.T) {
	_, err := MulDiv(maxUint256, maxUint256, new(uint256.Int), Floor)
	require.ErrorIs(t, err, ErrDivisionByZero)
	_, err = MulDiv(one, one, new(uint256.Int), Ceil)
	require.ErrorIs(t, err, ErrDivisionByZero)
}

func TestMulDiv_Overflow(t *testing.T) {
	_, err := MulDiv(maxUint256, u(2), one, Floor)
	require.ErrorIs(t, err, ErrOverflow)

	// (2^256 - 2^128 + 1) * (2^128 + 2) = (2^256 - 1) * (2^128 + 1) + 3, so the quotient is the
	// maximum, and rounding up the remainder overflows.
	x := new(uint256.Int).Sub(maxUint256, two128)
	x.AddUint64(x, 2)
	y := new(uint256.Int).AddUint64(two128, 2)
	denominator := new(uint256.Int).AddUint64(two128, 1)
	got, err := MulDiv(x, y, denominator, Floor)
	require.NoError(t, err)
	require.Equal(t, maxUint256, got)
	_, err = MulDiv(x, y, denominator, Ceil)
	require.ErrorIs(t, err, ErrOverflow)
}

// TestMulDiv_MatchesBigInt: random operands, against the quotient and remainder of the 512-bit
// product in math/big.
func TestMulDiv_MatchesBigInt(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 1))
	for range 20_000 {
		x, y, denominator := randomUint(r), randomUint(r), randomUint(r)
		floor, floorErr := MulDiv(x, y, denominator, Floor)
		ceil, ceilErr := MulDiv(x, y, denominator, Ceil)
		msg := x.Dec() + " * " + y.Dec() + " / " + denominator.Dec()
		if denominator.IsZero() {
			require.ErrorIs(t, floorErr, ErrDivisionByZero, msg)
			require.ErrorIs(t, ceilErr, ErrDivisionByZero, msg)
			continue
		}
		quotient, remainder := new(big.Int).QuoRem(
			new(big.Int).Mul(x.ToBig(), y.ToBig()), denominator.ToBig(), new(big.Int))
		requireFits(t, quotient, floor, floorErr, msg)
		if remainder.Sign() != 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
		requireFits(t, quotient, ceil, ceilErr, msg)
	}
}

func TestCeilDiv(t *testing.T) {
	for _, c := range []struct{ a, b, want *uint256.Int }{
		{u(7), u(2), u(4)},
		{u(8), u(2), u(4)},
		{u(0), u(2), u(0)},
		{maxUint256, one, maxUint256},
	} {
		got, err := CeilDiv(c.a, c.b)
		require.NoError(t, err)
		require.Equal(t, c.want, got, "%s / %s", c.a, c.b)
	}
	_, err := CeilDiv(one, new(uint256.Int))
	require.ErrorIs(t, err, ErrDivisionByZero)

	r := rand.New(rand.NewPCG(0x5eed, 2))
	for range 20_000 {
		a, b := randomUint(r), randomUint(r)
		if b.IsZero() {
			continue
		}
		want := new(big.Int).Add(a.ToBig(), b.ToBig())
		want.Sub(want, big.NewInt(1)).Quo(want, b.ToBig())
		got, err := CeilDiv(a, b)
		require.NoError(t, err)
		require.Equal(t, want.String(), got.Dec(), "%s / %s", a, b)
	}
}

func TestSqrt(t *testing.T) {
	for _, c := range []struct {
		a        *uint256.Int
		rounding Rounding
		want     *uint256.Int
	}{
		{u(0), Floor, u(0)},
		{u(1), Ceil, u(1)},
		{u(2), Floor, u(1)},
		{u(2), Ceil, u(2)},
		{u(16), Ceil, u(4)},
		{maxUint256, Floor, new(uint256.Int).SubUint64(two128, 1)},
		{maxUint256, Ceil, two128},
	} {
		require.Equal(t, c.want, Sqrt(c.a, c.rounding), "sqrt(%s)", c.a)
	}

	r := rand.New(rand.NewPCG(0x5eed, 3))
	for range 20_000 {
		a := randomUint(r)
		root := Sqrt(a, Floor).ToBig()
		next := new(big.Int).Add(root, big.NewInt(1))
		require.LessOrEqual(t, new(big.Int).Mul(root, root).Cmp(a.ToBig()), 0, "sqrt(%s)", a)
		require.Positive(t, new(big.Int).Mul(next, next).Cmp(a.ToBig()), "sqrt(%s)", a)
		want := root
		if new(big.Int).Mul(root, root).Cmp(a.ToBig()) < 0 {
			want = next
		}
		require.Equal(t, want.String(), Sqrt(a, Ceil).Dec(), "sqrt(%s)", a)
	}
}
