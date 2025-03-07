package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoRounding(t *testing.T) {
	res, err := muldiv(
		big.NewInt(6),
		big.NewInt(7),
		big.NewInt(2),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(21)))
}

func TestWithRounding(t *testing.T) {
	res, err := muldiv(
		big.NewInt(6),
		big.NewInt(7),
		big.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(11)))
}

func TestNoRoundingNeeded(t *testing.T) {
	res, err := muldiv(
		big.NewInt(8),
		big.NewInt(2),
		big.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(4)))
}

func TestDivideByZero(t *testing.T) {
	_, err := muldiv(
		big.NewInt(1),
		big.NewInt(1),
		new(big.Int),
		false,
	)
	require.Equal(t, err, ErrDivZero)
}

func TestOverflow(t *testing.T) {
	_, err := muldiv(
		U256Max,
		big.NewInt(2),
		One,
		false,
	)
	require.Equal(t, err, ErrOverflow)
}

func TestLargeNumbers(t *testing.T) {
	res, err := muldiv(
		IntFromString("123456789012345678901234567890"),
		IntFromString("987654321098765432109876543210"),
		One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("121932631137021795226185032733622923332237463801111263526900")))
}

func TestRoundingBehavior(t *testing.T) {
	x, y, d := big.NewInt(10), big.NewInt(10), big.NewInt(5)

	res, err := muldiv(x, y, d, true)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(20)))

	d = big.NewInt(6)

	res, err = muldiv(x, y, d, true)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(17)))
}

func TestZeroNumerator(t *testing.T) {
	res, err := muldiv(
		new(big.Int),
		big.NewInt(100),
		big.NewInt(10),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Sign())
}

func TestOneDenominator(t *testing.T) {
	res, err := muldiv(
		big.NewInt(123456789),
		big.NewInt(987654321),
		One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(121932631112635269)))
}

func TestMaxValuesNoRounding(t *testing.T) {
	res, err := muldiv(
		U256Max,
		One,
		One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(U256Max))
}

func TestMaxValuesWithRounding(t *testing.T) {
	res, err := muldiv(
		U256Max,
		One,
		One,
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(U256Max))
}

func TestRoundingUp(t *testing.T) {
	res, err := muldiv(
		U256Max,
		One,
		big.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("57896044618658097711785492504343953926634992332820282019728792003956564819968")))
}

func TestIntermediateOverflow(t *testing.T) {
	_, err := muldiv(
		U256Max,
		U256Max,
		One,
		false,
	)
	require.Equal(t, err, ErrOverflow)
}

func TestMaxValuesRoundingUpOverflow(t *testing.T) {
	res, err := muldiv(
		new(big.Int).Sub(U256Max, One),
		U256Max,
		new(big.Int).Sub(U256Max, One),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("115792089237316195423570985008687907853269984665640564039457584007913129639935")))
}

func TestRoundingEdgeCase(t *testing.T) {
	res, err := muldiv(
		big.NewInt(5),
		big.NewInt(5),
		big.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(13)))
}

func TestLargeIntermediateResult(t *testing.T) {
	res, err := muldiv(
		IntFromString("123456789012345678901234567890"),
		IntFromString("98765432109876543210987654321"),
		IntFromString("1219326311370217952261850327336229233322374638011112635269"),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(10)))
}

func TestSmallDenominatorLargeNumerator(t *testing.T) {
	res, err := muldiv(
		IntFromString("340282366920938463463374607431768211455"),
		big.NewInt(2),
		big.NewInt(3),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(IntFromString("226854911280625642308916404954512140970")))
}
