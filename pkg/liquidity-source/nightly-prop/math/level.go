package math

import "github.com/holiman/uint256"

var two = uint256.NewInt(2)

// PriceLevelConfig is NightlyPoolStorage.PriceLevelConfig: a level's offsets from the oracle mid
// price and its extra spreads, in millionths.
type PriceLevelConfig struct {
	LowerOffset    SignedSpread `json:"lowerOffset"`
	UpperOffset    SignedSpread `json:"upperOffset"`
	BidExtraSpread Spread       `json:"bidExtraSpread"`
	AskExtraSpread Spread       `json:"askExtraSpread"`
}

// Step is NightlyPoolMath.Step: one level's part of a quote. End is the fraction where the trade
// leaves the level.
type Step struct {
	BaseAmount  uint256.Int
	QuoteAmount uint256.Int
	End         Fraction
}

// AskRange is NightlyPoolMath.askRange: the ask prices at both level edges,
// mid * (1 + offset) * (1 + oracle ask spread + level ask spread). Exact.
func AskRange(config PriceLevelConfig, mid Price, oracleAskSpread Spread) (PriceRange, error) {
	spread, err := oracleAskSpread.Add(config.AskExtraSpread)
	if err != nil {
		return PriceRange{}, err
	}
	lower, err := mid.AddRelativeOffset(config.LowerOffset)
	if err == nil {
		lower, err = lower.AddRelative(spread)
	}
	if err != nil {
		return PriceRange{}, err
	}
	upper, err := mid.AddRelativeOffset(config.UpperOffset)
	if err == nil {
		upper, err = upper.AddRelative(spread)
	}
	if err != nil {
		return PriceRange{}, err
	}
	return PriceRange{Lower: lower, Upper: upper}, nil
}

// BidRange is NightlyPoolMath.bidRange: the bid prices at both level edges,
// mid * (1 + offset) * (1 - oracle bid spread - level bid spread). Exact.
func BidRange(config PriceLevelConfig, mid Price, oracleBidSpread Spread) (PriceRange, error) {
	spread, err := oracleBidSpread.Add(config.BidExtraSpread)
	if err != nil {
		return PriceRange{}, err
	}
	lower, err := mid.AddRelativeOffset(config.LowerOffset)
	if err == nil {
		lower, err = lower.SubRelative(spread)
	}
	if err != nil {
		return PriceRange{}, err
	}
	upper, err := mid.AddRelativeOffset(config.UpperOffset)
	if err == nil {
		upper, err = upper.SubRelative(spread)
	}
	if err != nil {
		return PriceRange{}, err
	}
	return PriceRange{Lower: lower, Upper: upper}, nil
}

// BuyWithExactQuote is NightlyPoolMath.buyWithExactQuote: it sells the level's base to the taker
// for exactly offeredQuote, or for less if that quote buys all of the base. The caller supplies a
// nonzero availableBase.
//
// The fraction moves by the base output, so the price rises linearly with base output, and the
// quote paid is the area under that line. With all of the available base valued at three prices
// (start, upper - start, and end), the end value solves
//
//	quoteAtEndPrice^2 = quoteAtStartPrice^2 + 2 * offeredQuote * extraQuoteAtUpperPrice
//
// and the base output is offeredQuote at the mean of the start and end prices. The taker receives
// base, so the start price and the quote amounts round up, and the base output rounds down.
func BuyWithExactQuote(availableBase *uint256.Int, start Fraction, offeredQuote *uint256.Int,
	ask PriceRange) (Step, error) {
	startPrice, err := ask.PriceAt(start, Ceil)
	if err != nil {
		return Step{}, err
	}
	// The most quote this level takes: all of its base at the mean of the start and upper prices.
	// Halving after quoteFor rounds the exact cost up once.
	priceSum, err := startPrice.Add(ask.Upper)
	if err != nil {
		return Step{}, err
	}
	quoteAtPriceSum, err := priceSum.QuoteFor(availableBase, Ceil)
	if err != nil {
		return Step{}, err
	}
	maxQuoteIn, err := CeilDiv(&quoteAtPriceSum, two)
	if err != nil {
		return Step{}, err
	}
	if !offeredQuote.Lt(&maxQuoteIn) {
		return Step{BaseAmount: *availableBase, QuoteAmount: maxQuoteIn, End: Upper}, nil
	}

	quoteAtStartPrice, err := startPrice.QuoteFor(availableBase, Ceil)
	if err != nil {
		return Step{}, err
	}
	extraPrice, err := ask.Upper.Sub(startPrice)
	if err != nil {
		return Step{}, err
	}
	extraQuoteAtUpperPrice, err := extraPrice.QuoteFor(availableBase, Ceil)
	if err != nil {
		return Step{}, err
	}
	twoOfferedQuote, err := Mul(two, offeredQuote)
	if err != nil {
		return Step{}, err
	}
	startSquare, err := Mul(&quoteAtStartPrice, &quoteAtStartPrice)
	if err != nil {
		return Step{}, err
	}
	growth, err := Mul(&twoOfferedQuote, &extraQuoteAtUpperPrice)
	if err != nil {
		return Step{}, err
	}
	endSquare, err := Add(&startSquare, &growth)
	if err != nil {
		return Step{}, err
	}
	quoteAtEndPrice := Sqrt(&endSquare, Ceil)
	// offeredQuote / mean(start price, end price), with both prices multiplied by availableBase.
	priceValueSum, err := Add(&quoteAtStartPrice, &quoteAtEndPrice)
	if err != nil {
		return Step{}, err
	}
	baseOut, err := MulDiv(&twoOfferedQuote, availableBase, &priceValueSum, Floor)
	if err != nil {
		return Step{}, err
	}
	end, err := start.MoveUp(&baseOut, availableBase, Floor)
	if err != nil {
		return Step{}, err
	}
	return Step{BaseAmount: baseOut, QuoteAmount: *offeredQuote, End: end}, nil
}

// SellExactBase is NightlyPoolMath.sellExactBase: it buys up to offeredBase from the taker with
// the level's quote. The caller supplies a nonzero availableQuote.
//
// The level's quote buys back at most maxBaseIn = availableQuote / refillBasePrice on the way to
// the lower edge, where refillBasePrice is the mean of the start and lower prices. If offeredBase
// reaches it, the step refills the level with base and takes all of its quote. Otherwise the
// fraction moves down by offeredBase / maxBaseIn of the distance to the lower edge, and the taker
// receives the base at the mean of the start and end prices. The taker receives quote, so prices,
// their means, and the quote output round down.
func SellExactBase(availableQuote *uint256.Int, start Fraction, offeredBase *uint256.Int,
	bid PriceRange) (Step, error) {
	startPrice, err := bid.PriceAt(start, Floor)
	if err != nil {
		return Step{}, err
	}
	refillBasePrice, err := startPrice.Mean(bid.Lower, Floor)
	if err != nil {
		return Step{}, err
	}
	// Rounded up, so that the taker pays at least the quote's value.
	maxBaseIn, err := refillBasePrice.BaseFor(availableQuote, Ceil)
	if err != nil {
		return Step{}, err
	}
	if !offeredBase.Lt(&maxBaseIn) {
		return Step{BaseAmount: maxBaseIn, QuoteAmount: *availableQuote, End: Lower}, nil
	}
	end, err := sellEnd(&start, offeredBase, availableQuote, &refillBasePrice)
	if err != nil {
		return Step{}, err
	}
	endPrice, err := bid.PriceAt(end, Floor)
	if err != nil {
		return Step{}, err
	}
	meanPrice, err := startPrice.Mean(endPrice, Floor)
	if err != nil {
		return Step{}, err
	}
	quoteOut, err := meanPrice.QuoteFor(offeredBase, Floor)
	if err != nil {
		return Step{}, err
	}
	return Step{BaseAmount: *offeredBase, QuoteAmount: quoteOut, End: end}, nil
}

// sellEnd is NightlyPoolMath._sellEnd: the position after a sale of offeredBase into a level whose
// quote buys back availableQuote / refillBasePrice base,
// start * (1 - offeredBase * refillBasePrice / availableQuote), without rounding the move to whole
// token units. Two mulDiv steps keep each intermediate value within uint256: the price factor
// comes first when it is at least one, and last when it is below one. Both round up, so the
// fraction moves at least the exact distance. The caller supplies offeredBase < maxBaseIn.
func sellEnd(start *Fraction, offeredBase, availableQuote *uint256.Int, refillBasePrice *Price) (Fraction, error) {
	raw := &start.raw
	var first, moved uint256.Int
	var err error
	if !refillBasePrice.Numerator.Lt(&refillBasePrice.Denominator) {
		if first, err = MulDiv(raw, &refillBasePrice.Numerator, &refillBasePrice.Denominator, Ceil); err == nil {
			moved, err = MulDiv(&first, offeredBase, availableQuote, Ceil)
		}
	} else {
		if first, err = MulDiv(raw, offeredBase, availableQuote, Ceil); err == nil {
			moved, err = MulDiv(&first, &refillBasePrice.Numerator, &refillBasePrice.Denominator, Ceil)
		}
	}
	if err != nil {
		return Fraction{}, err
	}
	// Rounding up twice can pass the lower edge only when the exact move is within a few units of it.
	capped := Min(&moved, raw)
	var rest uint256.Int
	rest.Sub(raw, &capped)
	return FractionFromRaw(&rest)
}
