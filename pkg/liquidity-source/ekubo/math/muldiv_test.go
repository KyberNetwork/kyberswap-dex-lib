package math

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestNoRounding(t *testing.T) {
	res, err := mulDivOverflow(
		big.NewInt(6),
		big.NewInt(7),
		big.NewInt(2),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(21)))
}

func TestWithRounding(t *testing.T) {
	res, err := mulDivOverflow(
		big.NewInt(6),
		big.NewInt(7),
		big.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(11)))
}

func TestNoRoundingNeeded(t *testing.T) {
	res, err := mulDivOverflow(
		big.NewInt(8),
		big.NewInt(2),
		big.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(4)))
}

func TestDivideByZero(t *testing.T) {
	_, err := mulDivOverflow(
		big.NewInt(1),
		big.NewInt(1),
		new(big.Int),
		false,
	)
	require.Equal(t, err, ErrDivZero)
}

func TestOverflow(t *testing.T) {
	_, err := mulDivOverflow(
		U256Max,
		big.NewInt(2),
		bignum.One,
		false,
	)
	require.Equal(t, err, ErrMulDivOverflow)
}

func TestLargeNumbers(t *testing.T) {
	res, err := mulDivOverflow(
		bignum.NewBig("123456789012345678901234567890"),
		bignum.NewBig("987654321098765432109876543210"),
		bignum.One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("121932631137021795226185032733622923332237463801111263526900")))
}

func TestRoundingBehavior(t *testing.T) {
	x, y, d := big.NewInt(10), big.NewInt(10), big.NewInt(5)

	res, err := mulDivOverflow(x, y, d, true)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(20)))

	d = big.NewInt(6)

	res, err = mulDivOverflow(x, y, d, true)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(17)))
}

func TestZeroNumerator(t *testing.T) {
	res, err := mulDivOverflow(
		new(big.Int),
		big.NewInt(100),
		big.NewInt(10),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Sign())
}

func TestOneDenominator(t *testing.T) {
	res, err := mulDivOverflow(
		big.NewInt(123456789),
		big.NewInt(987654321),
		bignum.One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(121932631112635269)))
}

func TestMaxValuesNoRounding(t *testing.T) {
	res, err := mulDivOverflow(
		U256Max,
		bignum.One,
		bignum.One,
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(U256Max))
}

func TestMaxValuesWithRounding(t *testing.T) {
	res, err := mulDivOverflow(
		U256Max,
		bignum.One,
		bignum.One,
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(U256Max))
}

func TestRoundingUp(t *testing.T) {
	res, err := mulDivOverflow(
		U256Max,
		bignum.One,
		big.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("57896044618658097711785492504343953926634992332820282019728792003956564819968")))
}

func TestIntermediateOverflow(t *testing.T) {
	_, err := mulDivOverflow(
		U256Max,
		U256Max,
		bignum.One,
		false,
	)
	require.Equal(t, err, ErrMulDivOverflow)
}

func TestMaxValuesRoundingUpOverflow(t *testing.T) {
	res, err := mulDivOverflow(
		new(big.Int).Sub(U256Max, bignum.One),
		U256Max,
		new(big.Int).Sub(U256Max, bignum.One),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("115792089237316195423570985008687907853269984665640564039457584007913129639935")))
}

func TestRoundingEdgeCase(t *testing.T) {
	res, err := mulDivOverflow(
		big.NewInt(5),
		big.NewInt(5),
		big.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(13)))
}

func TestLargeIntermediateResult(t *testing.T) {
	res, err := mulDivOverflow(
		bignum.NewBig("123456789012345678901234567890"),
		bignum.NewBig("98765432109876543210987654321"),
		bignum.NewBig("1219326311370217952261850327336229233322374638011112635269"),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(big.NewInt(10)))
}

func TestSmallDenominatorLargeNumerator(t *testing.T) {
	res, err := mulDivOverflow(
		bignum.NewBig("340282366920938463463374607431768211455"),
		big.NewInt(2),
		big.NewInt(3),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Cmp(bignum.NewBig("226854911280625642308916404954512140970")))
}
