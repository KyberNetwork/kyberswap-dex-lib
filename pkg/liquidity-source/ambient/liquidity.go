package ambient

import "math/big"

const (
	LotSizeBits      = 10
	LotActiveBits    = 11
	KnockoutFlagMask = 0x1
)

func LotsToLiquidity(lots *big.Int) *big.Int {
	realLots := new(big.Int).AndNot(lots, big.NewInt(KnockoutFlagMask))
	return new(big.Int).Lsh(realLots, LotSizeBits)
}

func LiquidityToLots(liq *big.Int) *big.Int {
	return new(big.Int).Rsh(liq, LotSizeBits)
}

func HasKnockoutLiq(lots *big.Int) bool {
	return new(big.Int).And(lots, big.NewInt(KnockoutFlagMask)).Sign() > 0
}

func NetLotsOnLiquidity(incrLots, decrLots *big.Int) *big.Int {
	incrLiq := LotsToLiquidity(incrLots)
	decrLiq := LotsToLiquidity(decrLots)
	return new(big.Int).Sub(incrLiq, decrLiq)
}

func ActiveLiquidity(curve *CurveState) *big.Int {
	ambient := InflateLiqSeed(curve.AmbientSeeds, curve.SeedDeflator)
	return new(big.Int).Add(ambient, curve.ConcLiq)
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

func BlendMileage(mileageX uint64, liqX *big.Int, mileageY uint64, liqY *big.Int) uint64 {
	if liqY.Sign() == 0 {
		return mileageX
	}
	if liqX.Sign() == 0 {
		return mileageY
	}
	if mileageX == mileageY {
		return mileageX
	}
	total := new(big.Int).Add(liqX, liqY)
	termX := calcBlend(mileageX, liqX, total)
	termY := calcBlend(mileageY, liqY, total)
	return (termX + 1) + (termY + 1)
}

func calcBlend(mileage uint64, weight, total *big.Int) uint64 {
	num := new(big.Int).Mul(new(big.Int).SetUint64(mileage), weight)
	result := num.Div(num, total)
	return result.Uint64()
}

func DeltaRewardsRate(feeMileage, oldMileage uint64) uint64 {
	const rewardRoundDown uint64 = 2
	if feeMileage > oldMileage+rewardRoundDown {
		return feeMileage - oldMileage - rewardRoundDown
	}
	return 0
}
