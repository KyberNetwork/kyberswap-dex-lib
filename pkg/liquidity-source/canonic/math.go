package canonic

import (
	"github.com/holiman/uint256"
)

var (
	uint256Ten = uint256.NewInt(10)
	uint256Two = uint256.NewInt(2)
)

func roundPriceToSigfigs(price *uint256.Int, sigfigs uint64) *uint256.Int {
	if sigfigs == 0 || price.IsZero() {
		return new(uint256.Int).Set(price)
	}

	numDigits := uint64(0)
	temp := new(uint256.Int).Set(price)
	for !temp.IsZero() {
		temp.Div(temp, uint256Ten)
		numDigits++
	}

	if numDigits <= sigfigs {
		return new(uint256.Int).Set(price)
	}

	exp := numDigits - sigfigs
	roundFactor := new(uint256.Int).Exp(uint256Ten, uint256.NewInt(exp))

	half := new(uint256.Int).Div(roundFactor, uint256Two)

	result := new(uint256.Int)
	result.Add(price, half)
	result.Div(result, roundFactor)
	result.Mul(result, roundFactor)

	return result
}

func ceilDiv(a, b *uint256.Int) *uint256.Int {
	var result uint256.Int
	bMinus1 := new(uint256.Int).Sub(b, uint256.NewInt(1))
	result.Add(a, bMinus1)
	result.Div(&result, b)
	return new(uint256.Int).Set(&result)
}

func calcBuyBaseTargetIn(
	quoteIn *uint256.Int,
	midPrice, midPrecision *uint256.Int,
	askRungs []uint16, askVolumes []*uint256.Int,
	takerFee, feeDenom *uint256.Int,
	rungDenom *uint256.Int,
	baseScale, quoteScale *uint256.Int,
	priceSigfigs uint64,
) (baseOut, fee *uint256.Int, err error) {
	var (
		remainingQuote uint256.Int
		totalBase      uint256.Int
		totalFee       uint256.Int
		rungPrice      uint256.Int
		baseAtRung     uint256.Int
		quoteForRung   uint256.Int
		rungBps        uint256.Int
		feeAtRung      uint256.Int
	)

	remainingQuote.Set(quoteIn)

	for i, rung := range askRungs {
		vol := askVolumes[i]
		if vol.IsZero() {
			continue
		}

		rungBps.SetUint64(uint64(rung))
		rungPrice.Add(rungDenom, &rungBps)
		rungPrice.Mul(&rungPrice, midPrice)
		rungPrice.Div(&rungPrice, rungDenom)

		if priceSigfigs > 0 {
			rounded := roundPriceToSigfigs(&rungPrice, priceSigfigs)
			rungPrice.Set(rounded)
		}

		quoteForRung.Mul(vol, &rungPrice)
		quoteForRung.Div(&quoteForRung, midPrecision)
		quoteForRung.Mul(&quoteForRung, quoteScale)
		quoteForRung.Div(&quoteForRung, baseScale)

		if remainingQuote.Cmp(&quoteForRung) >= 0 {
			totalBase.Add(&totalBase, vol)
			remainingQuote.Sub(&remainingQuote, &quoteForRung)
		} else {
			baseAtRung.Mul(&remainingQuote, midPrecision)
			baseAtRung.Div(&baseAtRung, &rungPrice)
			baseAtRung.Mul(&baseAtRung, baseScale)
			baseAtRung.Div(&baseAtRung, quoteScale)

			totalBase.Add(&totalBase, &baseAtRung)
			remainingQuote.Clear()
			break
		}
	}

	if totalBase.IsZero() {
		return nil, nil, ErrInsufficientLiquidity
	}

	totalFee.Mul(&totalBase, takerFee)
	totalFee.Set(ceilDiv(&totalFee, feeDenom))

	feeAtRung.Set(&totalFee)
	totalBase.Sub(&totalBase, &feeAtRung)

	baseOut = new(uint256.Int).Set(&totalBase)
	fee = new(uint256.Int).Set(&totalFee)
	return baseOut, fee, nil
}

func calcSellBaseTargetIn(
	baseIn *uint256.Int,
	midPrice, midPrecision *uint256.Int,
	bidRungs []uint16, bidVolumes []*uint256.Int,
	takerFee, feeDenom *uint256.Int,
	rungDenom *uint256.Int,
	baseScale, quoteScale *uint256.Int,
	priceSigfigs uint64,
) (quoteOut, fee *uint256.Int, err error) {
	var (
		remainingBase uint256.Int
		totalQuote    uint256.Int
		totalFee      uint256.Int
		rungPrice     uint256.Int
		quoteAtRung   uint256.Int
		baseCapacity  uint256.Int
		rungBps       uint256.Int
	)

	remainingBase.Set(baseIn)

	for i, rung := range bidRungs {
		vol := bidVolumes[i]
		if vol.IsZero() {
			continue
		}

		rungBps.SetUint64(uint64(rung))
		if rungBps.Cmp(rungDenom) >= 0 {
			continue
		}
		rungPrice.Sub(rungDenom, &rungBps)
		rungPrice.Mul(&rungPrice, midPrice)
		rungPrice.Div(&rungPrice, rungDenom)

		if priceSigfigs > 0 {
			rounded := roundPriceToSigfigs(&rungPrice, priceSigfigs)
			rungPrice.Set(rounded)
		}

		if rungPrice.IsZero() {
			continue
		}

		baseCapacity.Mul(vol, midPrecision)
		baseCapacity.Div(&baseCapacity, &rungPrice)
		baseCapacity.Mul(&baseCapacity, baseScale)
		baseCapacity.Div(&baseCapacity, quoteScale)

		if remainingBase.Cmp(&baseCapacity) >= 0 {
			totalQuote.Add(&totalQuote, vol)
			remainingBase.Sub(&remainingBase, &baseCapacity)
		} else {
			quoteAtRung.Mul(&remainingBase, &rungPrice)
			quoteAtRung.Div(&quoteAtRung, midPrecision)
			quoteAtRung.Mul(&quoteAtRung, quoteScale)
			quoteAtRung.Div(&quoteAtRung, baseScale)

			totalQuote.Add(&totalQuote, &quoteAtRung)
			remainingBase.Clear()
			break
		}
	}

	if totalQuote.IsZero() {
		return nil, nil, ErrInsufficientLiquidity
	}

	totalFee.Mul(&totalQuote, takerFee)
	totalFee.Set(ceilDiv(&totalFee, feeDenom))

	totalQuote.Sub(&totalQuote, &totalFee)

	quoteOut = new(uint256.Int).Set(&totalQuote)
	fee = new(uint256.Int).Set(&totalFee)
	return quoteOut, fee, nil
}

func estimateQuoteValue(
	baseAmount *uint256.Int,
	midPrice, midPrecision *uint256.Int,
	baseScale, quoteScale *uint256.Int,
) *uint256.Int {
	var result uint256.Int
	result.Mul(baseAmount, midPrice)
	result.Div(&result, midPrecision)
	result.Mul(&result, quoteScale)
	result.Div(&result, baseScale)
	return new(uint256.Int).Set(&result)
}

func parseVolumes(vols []string) []*uint256.Int {
	result := make([]*uint256.Int, len(vols))
	for i, v := range vols {
		val, err := uint256.FromDecimal(v)
		if err != nil {
			val = new(uint256.Int)
		}
		result[i] = val
	}
	return result
}
