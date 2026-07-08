package poe

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Reference values below were generated from the Python `get_quote` in
// native.md (verified bit-exact against on-chain getQuote), so these Go
// tests must match them exactly, to the last wei.

func mustBig(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}

func TestGetQuote_XtoY_NotCapped(t *testing.T) {
	reserveX := mustBig("10000000000000000000")    // 1e19
	reserveY := big.NewInt(20_000_000_000)         // 2e10
	price := big.NewInt(2_000_000_000_000_000_000) // 2e18
	fee := big.NewInt(3000)
	alpha := big.NewInt(10500)

	q, err := getQuote(reserveX, reserveY, big.NewInt(10_000_000_000), true, price, fee, alpha)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(18991), q.amountOut)
	require.Equal(t, big.NewInt(10_000_000_000), q.actualIn)
	require.Equal(t, big.NewInt(0), q.feeIn)
	require.Equal(t, big.NewInt(58), q.feeOut)
}

func TestGetQuote_XtoY_Capped(t *testing.T) {
	reserveX := mustBig("10000000000000000000")    // 1e19
	reserveY := big.NewInt(20_000_000_000)         // 2e10
	price := big.NewInt(2_000_000_000_000_000_000) // 2e18
	fee := big.NewInt(3000)
	alpha := big.NewInt(10500)

	q, err := getQuote(reserveX, reserveY, big.NewInt(100_000_000_000_000_000), true, price, fee, alpha)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(19_940_000_000), q.amountOut)
	actualIn, _ := new(big.Int).SetString("10499475576837990", 10)
	require.Equal(t, actualIn, q.actualIn)
	require.Equal(t, big.NewInt(0), q.feeIn)
	require.Equal(t, big.NewInt(60_000_000), q.feeOut)

	// partial fill: actualIn must be strictly less than the requested amount.
	require.True(t, q.actualIn.Cmp(big.NewInt(100_000_000_000_000_000)) < 0)
}

func TestGetQuote_YtoX_NotCapped(t *testing.T) {
	reserveX := mustBig("10000000000000000000")    // 1e19
	reserveY := big.NewInt(20_000_000_000)         // 2e10
	price := big.NewInt(2_000_000_000_000_000_000) // 2e18
	fee := big.NewInt(3000)
	alpha := big.NewInt(10500)

	q, err := getQuote(reserveX, reserveY, big.NewInt(5_000_000_000), false, price, fee, alpha)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(2_492_500_000_000_000), q.amountOut)
	require.Equal(t, big.NewInt(5_000_000_000), q.actualIn)
	require.Equal(t, big.NewInt(15_000_000), q.feeIn)
	require.Equal(t, big.NewInt(0), q.feeOut)
}

func TestGetQuote_YtoX_Capped(t *testing.T) {
	reserveX := mustBig("10000000000000000000")    // 1e19
	reserveY := big.NewInt(20_000_000_000)         // 2e10
	price := big.NewInt(2_000_000_000_000_000_000) // 2e18
	fee := big.NewInt(3000)
	alpha := big.NewInt(10500)

	q, err := getQuote(reserveX, reserveY, big.NewInt(100_000_000_000_000), false, price, fee, alpha)
	require.NoError(t, err)
	require.Equal(t, reserveX, q.amountOut) // fully drains the real reserve of X
	require.Equal(t, big.NewInt(20_308_122_082_975), q.actualIn)
	require.Equal(t, big.NewInt(60_924_366_249), q.feeIn)
	require.Equal(t, big.NewInt(0), q.feeOut)

	// partial fill: actualIn must be strictly less than the requested amount.
	require.True(t, q.actualIn.Cmp(big.NewInt(100_000_000_000_000)) < 0)
}

func TestGetQuote_ZeroAmountIn(t *testing.T) {
	_, err := getQuote(big.NewInt(1), big.NewInt(1), big.NewInt(0), true, big.NewInt(1), big.NewInt(0), big.NewInt(20000))
	require.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestFeeInclusive(t *testing.T) {
	require.Equal(t, big.NewInt(3000), feeInclusive(big.NewInt(1_000_000), big.NewInt(3000)))
	require.Equal(t, big.NewInt(3001), feeInclusive(big.NewInt(1_000_001), big.NewInt(3000)))
}

func TestFeeExclusive_RoundTrip(t *testing.T) {
	// feeExclusive(net, fee) should be the fee that, added on top of net,
	// makes feeInclusive(gross, fee) recover (approximately) that same fee.
	net := big.NewInt(1_000_000)
	fee := big.NewInt(3000)

	feeOnTop := feeExclusive(net, fee)
	gross := new(big.Int).Add(net, feeOnTop)

	recovered := feeInclusive(gross, fee)
	require.Equal(t, feeOnTop, recovered)
}
