package math

// The vectors of evm_contracts/test/NightlyPoolMath.t.sol for PriceMath.

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// p is a price with denominator one, so its numerator is the native price.
func p(numerator *uint256.Int) Price {
	return Price{Numerator: *numerator, Denominator: *one}
}

func pn(numerator uint64) Price {
	return p(u(numerator))
}

func pow10u(t *testing.T, exponent uint8) *uint256.Int {
	t.Helper()
	z, err := pow10(exponent)
	require.NoError(t, err)
	return z
}

// scaled returns value * 10^exponent.
func scaled(t *testing.T, value uint64, exponent uint8) *uint256.Int {
	t.Helper()
	return new(uint256.Int).Mul(u(value), pow10u(t, exponent))
}

func requireNumerator(t *testing.T, want *uint256.Int, got Price, err error) {
	t.Helper()
	require.NoError(t, err)
	require.Equal(t, want.Dec(), got.Numerator.Dec())
}

func TestPriceFromOracle(t *testing.T) {
	// ETH (18) / USDC (6) at 2,000: the gap goes into the denominator.
	price, err := PriceFromOracle(2_000e8, 18, 6)
	requireNumerator(t, scaled(t, 2_000, 36), price, err)
	require.Equal(t, pow10u(t, 48), &price.Denominator)
	// USDC (6) / WETH (18) at 0.0005: the gap goes into the numerator.
	price, err = PriceFromOracle(50_000, 6, 18)
	requireNumerator(t, scaled(t, 5, 44), price, err)
	require.Equal(t, pow10u(t, 36), &price.Denominator)
	price, err = PriceFromOracle(1e8, 8, 8)
	requireNumerator(t, pow10u(t, 36), price, err)
	require.Equal(t, pow10u(t, 36), &price.Denominator)
}

// TestPrice_AddRelativeSmallPrice: at $0.000001, a 0.01 bps spread gives ask $0.000001000001 and
// bid $0.000000999999. Rounding to eight decimals would widen the spread to 100 bps.
func TestPrice_AddRelativeSmallPrice(t *testing.T) {
	mid, err := PriceFromOracle(100, 18, 18)
	require.NoError(t, err)
	ask, err := mid.AddRelative(1)
	requireNumerator(t, scaled(t, 1_000_001, 24), ask, err)
	bid, err := mid.SubRelative(1)
	requireNumerator(t, scaled(t, 999_999, 24), bid, err)
}

// TestPrice_AddRelativeMaximumSpread: the adjustment uses 1e6 + spread, which needs more than the
// spread's 32 bits.
func TestPrice_AddRelativeMaximumSpread(t *testing.T) {
	price, err := pn(1e6).AddRelative(Spread(^uint32(0)))
	requireNumerator(t, u(4_295_967_295), price, err)
}

// TestPrice_AddRelativeInexact: 1 * 1.000001 is not an integer, so the adjustment fails instead of
// rounding.
func TestPrice_AddRelativeInexact(t *testing.T) {
	_, err := pn(1).AddRelative(1)
	require.ErrorIs(t, err, ErrInexactAdjustment)
}

func TestPrice_SubRelativeAboveWholePrice(t *testing.T) {
	_, err := pn(1e6).SubRelative(1_000_001)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestPrice_AddRelativeOffset(t *testing.T) {
	price, err := pn(1e6).AddRelativeOffset(-1_000)
	requireNumerator(t, u(999_000), price, err)
	price, err = pn(1e6).AddRelativeOffset(1_000)
	requireNumerator(t, u(1_001_000), price, err)
	_, err = pn(1e6).AddRelativeOffset(-1_000_001)
	require.ErrorIs(t, err, ErrOverflow)
}

// TestPriceRange_PriceAtFullFractionPrecision: one fraction unit must move the price. Reducing the
// fraction to E6 or E18 would lose it.
func TestPriceRange_PriceAtFullFractionPrecision(t *testing.T) {
	r := PriceRange{Lower: pn(10), Upper: p(new(uint256.Int).AddUint64(maxRaw, 10))}
	price, err := r.PriceAt(fraction(t, u(1)), Floor)
	requireNumerator(t, u(11), price, err)
}

func TestPriceRange_PriceAtRounding(t *testing.T) {
	r := PriceRange{Lower: pn(100), Upper: pn(103)}
	half := fraction(t, new(uint256.Int).Rsh(maxRaw, 1))
	price, err := r.PriceAt(half, Ceil)
	requireNumerator(t, u(102), price, err)
	price, err = r.PriceAt(half, Floor)
	requireNumerator(t, u(101), price, err)
}

// TestPriceRange_PriceAtWeightedEndpoints: interpolation equals the weighted endpoints
// (lower * (maxRaw - raw) + upper * raw) / maxRaw, rounded either way. Prices below 2^128 keep the
// weighted sum within uint256.
func TestPriceRange_PriceAtWeightedEndpoints(t *testing.T) {
	r := rand.New(rand.NewPCG(0x5eed, 6))
	for range 20_000 {
		lower, upper := randomBits(r, r.IntN(129)), randomBits(r, r.IntN(129))
		if upper.Lt(lower) {
			lower, upper = upper, lower
		}
		raw := randomBits(r, 104)
		denominator := randomUint(r)
		priceRange := PriceRange{
			Lower: Price{Numerator: *lower, Denominator: *denominator},
			Upper: Price{Numerator: *upper, Denominator: *denominator},
		}
		weighted := new(uint256.Int).Mul(lower, new(uint256.Int).Sub(maxRaw, raw))
		weighted.Add(weighted, new(uint256.Int).Mul(upper, raw))
		down, err := priceRange.PriceAt(fraction(t, raw), Floor)
		requireNumerator(t, new(uint256.Int).Div(weighted, maxRaw), down, err)
		up, err := priceRange.PriceAt(fraction(t, raw), Ceil)
		wantUp, ceilErr := CeilDiv(weighted, maxRaw)
		require.NoError(t, ceilErr)
		requireNumerator(t, wantUp, up, err)
		require.Equal(t, denominator, &down.Denominator)
		require.Equal(t, denominator, &up.Denominator)
	}
}

func TestPriceRange_PriceAtFullWidth(t *testing.T) {
	r := PriceRange{Lower: pn(0), Upper: p(maxUint256)}
	half := fraction(t, new(uint256.Int).Rsh(maxRaw, 1))
	down, err := r.PriceAt(half, Floor)
	require.NoError(t, err)
	up, err := r.PriceAt(half, Ceil)
	requireNumerator(t, new(uint256.Int).AddUint64(&down.Numerator, 1), up, err)
	lower, err := r.PriceAt(Lower, Ceil)
	requireNumerator(t, u(0), lower, err)
	upper, err := r.PriceAt(Upper, Floor)
	requireNumerator(t, maxUint256, upper, err)
}

func TestPriceRange_PriceAtError(t *testing.T) {
	_, err := PriceRange{
		Lower: Price{Numerator: *u(1), Denominator: *u(1)},
		Upper: Price{Numerator: *u(2), Denominator: *u(2)},
	}.PriceAt(Lower, Floor)
	require.ErrorIs(t, err, ErrDenominatorMismatch)
	// A negative width.
	_, err = PriceRange{Lower: pn(2), Upper: pn(1)}.PriceAt(Lower, Floor)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestPrice_Mean(t *testing.T) {
	price, err := pn(1).Mean(pn(2), Ceil)
	requireNumerator(t, u(2), price, err)
	price, err = pn(1).Mean(pn(2), Floor)
	requireNumerator(t, u(1), price, err)
	price, err = pn(2).Mean(pn(4), Ceil)
	requireNumerator(t, u(3), price, err)
}

func TestPrice_BaseFor(t *testing.T) {
	// 10 quote at price 3 is 3.33... base.
	base, err := pn(3).BaseFor(u(10), Ceil)
	require.NoError(t, err)
	require.Equal(t, u(4), base)
	base, err = pn(3).BaseFor(u(10), Floor)
	require.NoError(t, err)
	require.Equal(t, u(3), base)
	base, err = pn(1).BaseFor(maxUint256, Ceil)
	require.NoError(t, err)
	require.Equal(t, maxUint256, base)
	// At price 1 / 2, the maximum quote is worth twice the maximum base.
	_, err = Price{Numerator: *u(1), Denominator: *u(2)}.BaseFor(maxUint256, Ceil)
	require.ErrorIs(t, err, ErrOverflow)
}

func TestPrice_AddSub(t *testing.T) {
	sum, err := pn(5).Add(pn(3))
	requireNumerator(t, u(8), sum, err)
	difference, err := pn(5).Sub(pn(3))
	requireNumerator(t, u(2), difference, err)
	width, err := PriceRange{Lower: pn(100), Upper: pn(103)}.Width()
	requireNumerator(t, u(3), width, err)

	_, err = pn(3).Sub(pn(5))
	require.ErrorIs(t, err, ErrOverflow)
	_, err = pn(1).Add(Price{Numerator: *u(1), Denominator: *u(2)})
	require.ErrorIs(t, err, ErrDenominatorMismatch)
	_, err = pn(1).Mean(Price{Numerator: *u(1), Denominator: *u(2)}, Floor)
	require.ErrorIs(t, err, ErrDenominatorMismatch)
}

func TestPrice_RepriceQuote(t *testing.T) {
	// 10 quote at price 3 buys 3.33... base, which is worth 6.66... quote at price 2.
	quote, err := pn(3).RepriceQuote(u(10), pn(2), Ceil)
	require.NoError(t, err)
	require.Equal(t, u(7), quote)
	quote, err = pn(3).RepriceQuote(u(10), pn(2), Floor)
	require.NoError(t, err)
	require.Equal(t, u(6), quote)
	_, err = pn(1).RepriceQuote(u(1), Price{Numerator: *u(1), Denominator: *u(2)}, Floor)
	require.ErrorIs(t, err, ErrDenominatorMismatch)
}

func TestPrice_Mul(t *testing.T) {
	half := fraction(t, new(uint256.Int).Rsh(maxRaw, 1))
	price, err := pn(3).Mul(half, Floor)
	requireNumerator(t, u(1), price, err)
	price, err = pn(3).Mul(half, Ceil)
	requireNumerator(t, u(2), price, err)
	price, err = pn(3).Mul(Upper, Floor)
	requireNumerator(t, u(3), price, err)
}
