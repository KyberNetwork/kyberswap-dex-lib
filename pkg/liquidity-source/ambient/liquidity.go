package ambient

import "math/big"

const (
	LotSizeBits = 10
)

var (
	knockoutFlagMask = big.NewInt(0x1)
)

func LotsToLiquidity(lots *big.Int) *big.Int {
	result := new(big.Int).AndNot(lots, knockoutFlagMask)
	return result.Lsh(result, LotSizeBits)
}

func HasKnockoutLiq(lots *big.Int) bool {
	return lots != nil && lots.Bit(0) == 1
}

func ActiveLiquidity(curve *CurveState) *big.Int {
	ambient := InflateLiqSeed(curve.AmbientSeeds, curve.SeedDeflator)
	return ambient.Add(ambient, curve.ConcLiq)
}

func DeltaBase(liq, priceX, priceY *big.Int) *big.Int {
	priceDelta := new(big.Int).Sub(priceX, priceY)
	if priceDelta.Sign() < 0 {
		priceDelta.Neg(priceDelta)
	}
	return ReserveAtPrice(liq, priceDelta, true)
}

func DeltaQuote(liq, price, limitPrice *big.Int) *big.Int {
	if limitPrice.Cmp(price) > 0 {
		return calcQuoteDelta(liq, limitPrice, price)
	}
	return calcQuoteDelta(liq, price, limitPrice)
}

func calcQuoteDelta(liq, priceBig, priceSmall *big.Int) *big.Int {
	priceDelta := new(big.Int).Sub(priceBig, priceSmall)
	termOne := DivQ64(liq, priceSmall)
	termTwo := new(big.Int).Mul(termOne, priceDelta)
	termTwo.Div(termTwo, priceBig)
	return termTwo
}

func ReserveAtPrice(liq, price *big.Int, inBaseQty bool) *big.Int {
	if inBaseQty {
		return MulQ64(liq, price)
	}
	return DivQ64(liq, price)
}
