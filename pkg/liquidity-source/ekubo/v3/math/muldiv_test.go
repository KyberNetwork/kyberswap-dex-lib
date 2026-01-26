package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestNoRounding(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		uint256.NewInt(6),
		uint256.NewInt(7),
		uint256.NewInt(2),
		false,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(21), res)
}

func TestWithRounding(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		uint256.NewInt(6),
		uint256.NewInt(7),
		uint256.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(11), res)
}

func TestNoRoundingNeeded(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		uint256.NewInt(8),
		uint256.NewInt(2),
		uint256.NewInt(4),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(4), res)
}

func TestDivideByZero(t *testing.T) {
	t.Parallel()
	_, err := MulDivOverflow(
		uint256.NewInt(1),
		uint256.NewInt(1),
		new(uint256.Int),
		false,
	)
	require.ErrorIs(t, err, ErrDivZero)
}

func TestLargeNumbers(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.New("123456789012345678901234567890"),
		big256.New("987654321098765432109876543210"),
		big256.U1,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("121932631137021795226185032733622923332237463801111263526900"), res)
}

func TestRoundingBehavior(t *testing.T) {
	t.Parallel()
	x, y, d := uint256.NewInt(10), uint256.NewInt(10), uint256.NewInt(5)

	res, err := MulDivOverflow(x, y, d, true)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(20), res)

	d = uint256.NewInt(6)

	res, err = MulDivOverflow(x, y, d, true)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(17), res)
}

func TestZeroNumerator(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		new(uint256.Int),
		uint256.NewInt(100),
		uint256.NewInt(10),
		false,
	)
	require.NoError(t, err)
	require.Zero(t, res.Sign())
}

func TestOneDenominator(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		uint256.NewInt(123456789),
		uint256.NewInt(987654321),
		big256.U1,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(121932631112635269), res)
}

func TestMaxValuesNoRounding(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.UMax,
		big256.U1,
		big256.U1,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, big256.UMax, res)
}

func TestMaxValuesWithRounding(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.UMax,
		big256.U1,
		big256.U1,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, big256.UMax, res)
}

func TestRoundingUp(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.UMax,
		big256.U1,
		uint256.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("57896044618658097711785492504343953926634992332820282019728792003956564819968"), res)
}

func TestMaxValuesRoundingUpOverflow(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		new(uint256.Int).Sub(big256.UMax, big256.U1),
		big256.UMax,
		new(uint256.Int).Sub(big256.UMax, big256.U1),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("115792089237316195423570985008687907853269984665640564039457584007913129639935"), res)
}

func TestRoundingEdgeCase(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		uint256.NewInt(5),
		uint256.NewInt(5),
		uint256.NewInt(2),
		true,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(13), res)
}

func TestLargeIntermediateResult(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.New("123456789012345678901234567890"),
		big256.New("98765432109876543210987654321"),
		big256.New("1219326311370217952261850327336229233322374638011112635269"),
		false,
	)
	require.NoError(t, err)
	require.Equal(t, uint256.NewInt(10), res)
}

func TestSmallDenominatorLargeNumerator(t *testing.T) {
	t.Parallel()
	res, err := MulDivOverflow(
		big256.New("340282366920938463463374607431768211455"),
		uint256.NewInt(2),
		uint256.NewInt(3),
		false,
	)
	require.NoError(t, err)
	require.Equal(t, big256.New("226854911280625642308916404954512140970"), res)
}
