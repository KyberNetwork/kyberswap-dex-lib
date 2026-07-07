package ambient

import "github.com/holiman/uint256"

const LotSizeBits = 10

// LotsToLiquidity sets dst = (lots & ^1) << LotSizeBits and returns dst.
func LotsToLiquidity(dst, lots *uint256.Int) *uint256.Int {
	dst.And(lots, uKnockoutClearMask)
	return dst.Lsh(dst, LotSizeBits)
}

// HasKnockoutLiq returns true when bit 0 of lots is set (knockout flag).
func HasKnockoutLiq(lots *uint256.Int) bool {
	return lots != nil && lots[0]&1 == 1
}

// ActiveLiquidity sets dst = InflateLiqSeed(ambientSeeds, deflator) + concLiq.
func ActiveLiquidity(dst *uint256.Int, curve *CurveState) *uint256.Int {
	InflateLiqSeed(dst, &curve.AmbientSeeds, curve.SeedDeflator)
	return dst.Add(dst, &curve.ConcLiq)
}

// DeltaBase sets dst = MulQ64(liq, |priceX - priceY|).
func DeltaBase(dst, liq, priceX, priceY *uint256.Int) *uint256.Int {
	var delta uint256.Int
	if priceX.Gt(priceY) {
		delta.Sub(priceX, priceY)
	} else {
		delta.Sub(priceY, priceX)
	}
	return MulQ64(dst, liq, &delta)
}

// DeltaQuote sets dst = delta(liq, price, limitPrice) (Q64 reserve difference).
func DeltaQuote(dst, liq, price, limitPrice *uint256.Int) *uint256.Int {
	if limitPrice.Gt(price) {
		return calcQuoteDelta(dst, liq, limitPrice, price)
	}
	return calcQuoteDelta(dst, liq, price, limitPrice)
}

func calcQuoteDelta(dst, liq, priceBig, priceSmall *uint256.Int) *uint256.Int {
	var termOne, delta uint256.Int
	DivQ64(&termOne, liq, priceSmall)
	delta.Sub(priceBig, priceSmall)
	dst.Mul(&termOne, &delta)
	return dst.Div(dst, priceBig)
}

// ReserveAtPrice sets dst = MulQ64(liq, price) for base, DivQ64(liq, price) for quote.
func ReserveAtPrice(dst, liq, price *uint256.Int, inBaseQty bool) *uint256.Int {
	if inBaseQty {
		return MulQ64(dst, liq, price)
	}
	return DivQ64(dst, liq, price)
}

// uKnockoutClearMask is ^uint256(1), used in LotsToLiquidity.
var uKnockoutClearMask = new(uint256.Int).Not(new(uint256.Int).SetUint64(1))
