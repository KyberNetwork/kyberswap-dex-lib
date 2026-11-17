package math

import "github.com/holiman/uint256"

var (
	ten = uint256.NewInt(10)
	// internalPriceScale is PriceMath.INTERNAL_PRICE_SCALE: whole-token prices use 36 decimals
	// before the token decimals apply.
	internalPriceScale = new(uint256.Int).Exp(ten, uint256.NewInt(36))
	// oracleToInternal is PriceMath.ORACLE_TO_INTERNAL: it converts the oracle's eight decimals to
	// internalPriceScale.
	oracleToInternal  = new(uint256.Int).Exp(ten, uint256.NewInt(28))
	spreadDenominator = uint256.NewInt(SpreadDenominator)
)

// Price is the pool's Price: native quote units per native base unit, Numerator / Denominator.
// All prices of one pool share one denominator, so operations combine numerators directly. An
// operation on two prices with different denominators returns ErrDenominatorMismatch.
type Price struct {
	Numerator   uint256.Int
	Denominator uint256.Int
}

// PriceRange is the pool's PriceRange: the prices at the lower and upper edges of one level, with
// the same denominator.
type PriceRange struct {
	Lower Price
	Upper Price
}

// PriceFromOracle is PriceMath.fromOracle: an oracle midpoint, in quote tokens per base token with
// eight decimals, as a native Price in quote units per base unit. Exact for 0 to 18 decimals per
// token.
func PriceFromOracle(mid uint64, baseDecimals, quoteDecimals uint8) (Price, error) {
	// mid is below 2^64, so the product fits uint256.
	numerator := new(uint256.Int).Mul(uint256.NewInt(mid), oracleToInternal)
	if quoteDecimals > baseDecimals {
		// Example: USDC (6) / DAI (18) gives book price * 1e36 * 10^12 over 1e36.
		scale, err := pow10(quoteDecimals - baseDecimals)
		if err != nil {
			return Price{}, err
		}
		scaled, err := Mul(numerator, scale)
		if err != nil {
			return Price{}, err
		}
		return Price{Numerator: *scaled, Denominator: *internalPriceScale}, nil
	}
	// Example: ETH (18) / USDC (6) gives book price * 1e36 over 1e36 * 10^12.
	scale, err := pow10(baseDecimals - quoteDecimals)
	if err != nil {
		return Price{}, err
	}
	denominator, err := Mul(internalPriceScale, scale)
	if err != nil {
		return Price{}, err
	}
	return Price{Numerator: *numerator, Denominator: *denominator}, nil
}

// pow10 is Solidity's checked 10 ** exponent: a power above uint256 is ErrOverflow.
func pow10(exponent uint8) (*uint256.Int, error) {
	z := uint256.NewInt(1)
	for range exponent {
		var err error
		if z, err = Mul(z, ten); err != nil {
			return nil, err
		}
	}
	return z, nil
}

// AddRelativeOffset is PriceMath.addRelative(Price, SignedSpread): price + price * offset, the
// price moved by a signed fraction of itself. Exact: a result that would round is
// ErrInexactAdjustment. An offset below -100% is ErrOverflow, as SafeCast.toUint256 reverts.
func (p Price) AddRelativeOffset(offset SignedSpread) (Price, error) {
	spreadNumerator := int64(SpreadDenominator) + int64(offset)
	if spreadNumerator < 0 {
		return Price{}, ErrOverflow
	}
	return p.applySpread(uint256.NewInt(uint64(spreadNumerator)))
}

// AddRelative is PriceMath.addRelative(Price, Spread): price + price * spread, for example an ask
// above its reference. Exact: a result that would round is ErrInexactAdjustment.
func (p Price) AddRelative(spread Spread) (Price, error) {
	return p.applySpread(uint256.NewInt(SpreadDenominator + uint64(spread)))
}

// SubRelative is PriceMath.subRelative: price - price * spread, for example a bid below its
// reference. Exact: a result that would round is ErrInexactAdjustment. A spread above 100% is
// ErrOverflow, as the checked subtraction reverts.
func (p Price) SubRelative(spread Spread) (Price, error) {
	if uint64(spread) > SpreadDenominator {
		return Price{}, ErrOverflow
	}
	return p.applySpread(uint256.NewInt(SpreadDenominator - uint64(spread)))
}

// applySpread is PriceMath._applySpread: price * spreadNumerator / SpreadDenominator, where
// spreadNumerator is SpreadDenominator plus the spread or offset. It returns ErrInexactAdjustment
// instead of rounding.
func (p Price) applySpread(spreadNumerator *uint256.Int) (Price, error) {
	product, err := Mul(&p.Numerator, spreadNumerator)
	if err != nil {
		return Price{}, err
	}
	numerator, remainder := new(uint256.Int).DivMod(product, spreadDenominator, new(uint256.Int))
	if !remainder.IsZero() {
		return Price{}, ErrInexactAdjustment
	}
	return Price{Numerator: *numerator, Denominator: p.Denominator}, nil
}

// PriceAt is PriceMath.priceAt: the price at fraction of the way through the range, lower +
// (upper - lower) * fraction. One mulDiv by maxRaw rounds the move, so the fraction's full
// precision applies.
func (r PriceRange) PriceAt(fraction Fraction, rounding Rounding) (Price, error) {
	difference, err := Sub(&r.Upper.Numerator, &r.Lower.Numerator)
	if err != nil {
		return Price{}, err
	}
	if err := sharedDenominator(r.Upper, r.Lower); err != nil {
		return Price{}, err
	}
	offset, err := MulDiv(difference, &fraction.raw, maxRaw, rounding)
	if err != nil {
		return Price{}, err
	}
	numerator, err := Add(&r.Lower.Numerator, offset)
	if err != nil {
		return Price{}, err
	}
	return Price{Numerator: *numerator, Denominator: r.Lower.Denominator}, nil
}

// Width is PriceMath.width: the price distance from the lower edge to the upper edge.
func (r PriceRange) Width() (Price, error) {
	return r.Upper.Sub(r.Lower)
}

// Add is PriceMath.add.
func (p Price) Add(other Price) (Price, error) {
	if err := sharedDenominator(p, other); err != nil {
		return Price{}, err
	}
	numerator, err := Add(&p.Numerator, &other.Numerator)
	if err != nil {
		return Price{}, err
	}
	return Price{Numerator: *numerator, Denominator: p.Denominator}, nil
}

// Sub is PriceMath.sub. A negative price is ErrOverflow, as the checked subtraction reverts.
func (p Price) Sub(other Price) (Price, error) {
	if err := sharedDenominator(p, other); err != nil {
		return Price{}, err
	}
	numerator, err := Sub(&p.Numerator, &other.Numerator)
	if err != nil {
		return Price{}, err
	}
	return Price{Numerator: *numerator, Denominator: p.Denominator}, nil
}

// Mul is PriceMath.mul: price * fraction, where Lower is zero and Upper is one.
func (p Price) Mul(fraction Fraction, rounding Rounding) (Price, error) {
	numerator, err := MulDiv(&p.Numerator, &fraction.raw, maxRaw, rounding)
	if err != nil {
		return Price{}, err
	}
	return Price{Numerator: *numerator, Denominator: p.Denominator}, nil
}

// Mean is PriceMath.mean: the arithmetic mean of two prices.
func (p Price) Mean(other Price, rounding Rounding) (Price, error) {
	if err := sharedDenominator(p, other); err != nil {
		return Price{}, err
	}
	sum, err := Add(&p.Numerator, &other.Numerator)
	if err != nil {
		return Price{}, err
	}
	odd := sum.Uint64()&1 == 1
	numerator := sum.Rsh(sum, 1)
	if rounding == Ceil && odd {
		numerator.AddUint64(numerator, 1)
	}
	return Price{Numerator: *numerator, Denominator: p.Denominator}, nil
}

// QuoteFor is PriceMath.quoteFor: a native base amount as a native quote amount at the price.
func (p Price) QuoteFor(baseAmount *uint256.Int, rounding Rounding) (*uint256.Int, error) {
	return MulDiv(baseAmount, &p.Numerator, &p.Denominator, rounding)
}

// BaseFor is PriceMath.baseFor: a native quote amount as a native base amount at the price. A base
// amount above uint256 is ErrOverflow.
func (p Price) BaseFor(quoteAmount *uint256.Int, rounding Rounding) (*uint256.Int, error) {
	return MulDiv(quoteAmount, &p.Denominator, &p.Numerator, rounding)
}

// RepriceQuote is PriceMath.repriceQuote: a native quote amount at the price as the quote amount of
// the same base at newPrice, quoteAmount * newPrice / price, without rounding the base between the
// two prices.
func (p Price) RepriceQuote(quoteAmount *uint256.Int, newPrice Price, rounding Rounding) (*uint256.Int, error) {
	if err := sharedDenominator(p, newPrice); err != nil {
		return nil, err
	}
	return MulDiv(quoteAmount, &newPrice.Numerator, &p.Numerator, rounding)
}

// sharedDenominator is PriceMath._sharedDenominator's check: the two prices must share their
// denominator, so that their numerators can be combined directly.
func sharedDenominator(a, b Price) error {
	if !a.Denominator.Eq(&b.Denominator) {
		return ErrDenominatorMismatch
	}
	return nil
}
