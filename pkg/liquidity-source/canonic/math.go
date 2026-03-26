package canonic

import (
	"github.com/holiman/uint256"
)

func ceilDiv(a, b *uint256.Int) *uint256.Int {
	if b.IsZero() {
		return new(uint256.Int)
	}
	res := new(uint256.Int)
	mod := new(uint256.Int)
	res.DivMod(a, b, mod)
	if !mod.IsZero() {
		res.AddUint64(res, 1)
	}
	return res
}

func roundPriceQ(priceQ *uint256.Int, sigfigs uint32) *uint256.Int {
	if priceQ.IsZero() {
		return new(uint256.Int)
	}
	digits := countDigits(priceQ)
	if digits <= sigfigs {
		return new(uint256.Int).Set(priceQ)
	}
	exp := digits - sigfigs
	scale := pow10(exp)
	halfScale := new(uint256.Int).Rsh(scale, 1)
	rounded := new(uint256.Int).Add(priceQ, halfScale)
	rounded.Div(rounded, scale)
	rounded.Mul(rounded, scale)
	return rounded
}

func countDigits(n *uint256.Int) uint32 {
	if n.IsZero() {
		return 1
	}
	var count uint32 = 1
	tmp := new(uint256.Int).Set(n)
	ten := uint256.NewInt(10)
	for tmp.Cmp(ten) >= 0 {
		tmp.Div(tmp, ten)
		count++
	}
	return count
}

func pow10(exp uint32) *uint256.Int {
	result := uint256.NewInt(1)
	base := uint256.NewInt(10)
	e := exp
	for e > 0 {
		if e&1 == 1 {
			result.Mul(result, base)
		}
		base.Mul(base, new(uint256.Int).Set(base))
		e >>= 1
	}
	return result
}

func calcAskRungPrice(midPrice, midPrec *uint256.Int, rungBps uint16, quoteScale *uint256.Int) *uint256.Int {
	rungPrice := new(uint256.Int).Mul(midPrice, new(uint256.Int).Add(rungDenom, uint256.NewInt(uint64(rungBps))))
	rungPrice.Div(rungPrice, rungDenom)
	priceQ := new(uint256.Int).Mul(rungPrice, quoteScale)
	priceQ.Div(priceQ, midPrec)
	return roundPriceQ(priceQ, priceSigfigs)
}

func calcBidRungPrice(midPrice, midPrec *uint256.Int, rungBps uint16, quoteScale *uint256.Int) *uint256.Int {
	sub := new(uint256.Int).Sub(rungDenom, uint256.NewInt(uint64(rungBps)))
	rungPrice := new(uint256.Int).Mul(midPrice, sub)
	rungPrice.Div(rungPrice, rungDenom)
	priceQ := new(uint256.Int).Mul(rungPrice, quoteScale)
	priceQ.Div(priceQ, midPrec)
	return roundPriceQ(priceQ, priceSigfigs)
}

func calcSellBaseTargetIn(
	baseAmountIn *uint256.Int,
	midPrice, midPrec, takerFee, baseScale, quoteScale *uint256.Int,
	bidBps []uint16,
	bidVols []*uint256.Int,
) (quoteOut, quoteFee, baseUsed *uint256.Int) {
	remaining := new(uint256.Int).Set(baseAmountIn)
	totalQuoteGross := new(uint256.Int)

	for i := range bidBps {
		if remaining.IsZero() {
			break
		}
		vol := bidVols[i]
		if vol.IsZero() {
			continue
		}
		priceQ := calcBidRungPrice(midPrice, midPrec, bidBps[i], quoteScale)
		if priceQ.IsZero() {
			continue
		}

		baseCapacity := new(uint256.Int).Mul(vol, baseScale)
		baseCapacity.Div(baseCapacity, priceQ)

		baseFill := new(uint256.Int).Set(remaining)
		if baseFill.Cmp(baseCapacity) > 0 {
			baseFill = baseCapacity
		}

		quoteGross := new(uint256.Int).Mul(baseFill, priceQ)
		quoteGross.Div(quoteGross, baseScale)

		totalQuoteGross.Add(totalQuoteGross, quoteGross)
		remaining = new(uint256.Int).Sub(remaining, baseFill)
	}

	if totalQuoteGross.IsZero() {
		return new(uint256.Int), new(uint256.Int), new(uint256.Int)
	}

	fee := ceilDiv(new(uint256.Int).Mul(totalQuoteGross, takerFee), feeDenom)
	netQuote := new(uint256.Int).Sub(totalQuoteGross, fee)
	used := new(uint256.Int).Sub(baseAmountIn, remaining)

	return netQuote, fee, used
}

func calcBuyBaseTargetIn(
	quoteAmountIn *uint256.Int,
	midPrice, midPrec, takerFee, baseScale, quoteScale *uint256.Int,
	askBps []uint16,
	askVols []*uint256.Int,
) (baseOut, baseFee, quoteUsed *uint256.Int) {
	remaining := new(uint256.Int).Set(quoteAmountIn)
	totalBaseGross := new(uint256.Int)

	for i := range askBps {
		if remaining.IsZero() {
			break
		}
		vol := askVols[i]
		if vol.IsZero() {
			continue
		}
		priceQ := calcAskRungPrice(midPrice, midPrec, askBps[i], quoteScale)
		if priceQ.IsZero() {
			continue
		}

		quoteCost := new(uint256.Int).Mul(vol, priceQ)
		quoteCost.Div(quoteCost, baseScale)

		var baseFill, quoteSpent *uint256.Int
		if remaining.Cmp(quoteCost) >= 0 {
			baseFill = new(uint256.Int).Set(vol)
			quoteSpent = quoteCost
		} else {
			baseFill = new(uint256.Int).Mul(remaining, baseScale)
			baseFill.Div(baseFill, priceQ)
			quoteSpent = new(uint256.Int).Set(remaining)
		}

		totalBaseGross.Add(totalBaseGross, baseFill)
		remaining = new(uint256.Int).Sub(remaining, quoteSpent)
	}

	if totalBaseGross.IsZero() {
		return new(uint256.Int), new(uint256.Int), new(uint256.Int)
	}

	fee := ceilDiv(new(uint256.Int).Mul(totalBaseGross, takerFee), feeDenom)
	netBase := new(uint256.Int).Sub(totalBaseGross, fee)
	used := new(uint256.Int).Sub(quoteAmountIn, remaining)

	return netBase, fee, used
}

func calcSellBaseAmountIn(
	quoteDesired *uint256.Int,
	midPrice, midPrec, takerFee, baseScale, quoteScale *uint256.Int,
	bidBps []uint16,
	bidVols []*uint256.Int,
) (baseNeeded, quoteFee *uint256.Int) {
	grossNeeded := ceilDiv(
		new(uint256.Int).Mul(quoteDesired, feeDenom),
		new(uint256.Int).Sub(feeDenom, takerFee),
	)
	remaining := new(uint256.Int).Set(grossNeeded)
	totalBase := new(uint256.Int)
	quoteAccum := new(uint256.Int)

	for i := range bidBps {
		if remaining.IsZero() {
			break
		}
		vol := bidVols[i]
		if vol.IsZero() {
			continue
		}
		priceQ := calcBidRungPrice(midPrice, midPrec, bidBps[i], quoteScale)
		if priceQ.IsZero() {
			continue
		}

		quoteFill := new(uint256.Int).Set(vol)
		if quoteFill.Cmp(remaining) > 0 {
			quoteFill = new(uint256.Int).Set(remaining)
		}

		baseCost := ceilDiv(
			new(uint256.Int).Mul(quoteFill, baseScale),
			priceQ,
		)

		totalBase.Add(totalBase, baseCost)
		quoteAccum.Add(quoteAccum, quoteFill)
		remaining = new(uint256.Int).Sub(remaining, quoteFill)
	}

	if totalBase.IsZero() || !remaining.IsZero() {
		return nil, nil
	}

	fee := ceilDiv(new(uint256.Int).Mul(quoteAccum, takerFee), feeDenom)

	return totalBase, fee
}

func calcBuyBaseAmountIn(
	baseDesired *uint256.Int,
	midPrice, midPrec, takerFee, baseScale, quoteScale *uint256.Int,
	askBps []uint16,
	askVols []*uint256.Int,
) (quoteNeeded, baseFee *uint256.Int) {
	grossNeeded := ceilDiv(
		new(uint256.Int).Mul(baseDesired, feeDenom),
		new(uint256.Int).Sub(feeDenom, takerFee),
	)
	remaining := new(uint256.Int).Set(grossNeeded)
	totalQuote := new(uint256.Int)

	for i := range askBps {
		if remaining.IsZero() {
			break
		}
		vol := askVols[i]
		if vol.IsZero() {
			continue
		}
		priceQ := calcAskRungPrice(midPrice, midPrec, askBps[i], quoteScale)
		if priceQ.IsZero() {
			continue
		}

		baseFill := new(uint256.Int).Set(vol)
		if baseFill.Cmp(remaining) > 0 {
			baseFill = new(uint256.Int).Set(remaining)
		}

		quoteCost := ceilDiv(
			new(uint256.Int).Mul(baseFill, priceQ),
			baseScale,
		)

		totalQuote.Add(totalQuote, quoteCost)
		remaining = new(uint256.Int).Sub(remaining, baseFill)
	}

	if totalQuote.IsZero() || !remaining.IsZero() {
		return nil, nil
	}

	fee := ceilDiv(new(uint256.Int).Mul(grossNeeded, takerFee), feeDenom)

	return totalQuote, fee
}
