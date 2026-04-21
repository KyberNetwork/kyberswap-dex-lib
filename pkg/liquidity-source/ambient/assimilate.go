package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func AssimilateLiq(curve *CurveState, feesPaid *big.Int, isSwapInBase bool) {
	liq := ActiveLiquidity(curve)
	if liq.Sign() == 0 {
		return
	}
	feesInBase := !isSwapInBase
	feesToLiq := shaveForPrecision(liq, curve.PriceRoot, feesPaid, feesInBase)
	if feesToLiq.Sign() == 0 {
		return
	}
	inflator := calcLiqInflator(liq, curve.PriceRoot, feesToLiq, feesInBase)
	if inflator > 0 {
		stepToLiquidity(curve, inflator, feesInBase)
	}
}

func calcLiqInflator(liq, price, feesPaid *big.Int, inBaseQty bool) uint64 {
	reserve := ReserveAtPrice(liq, price, inBaseQty)
	return calcReserveInflator(reserve, feesPaid)
}

func calcReserveInflator(reserve, feesPaid *big.Int) uint64 {
	if reserve.Sign() == 0 || feesPaid.Cmp(reserve) > 0 {
		return 0
	}
	nextReserve := new(big.Int).Add(reserve, feesPaid)
	inflatorRoot := CompoundDivide(nextReserve, reserve)
	return ApproxSqrtCompound(inflatorRoot)
}

func shaveForPrecision(liq, price, feesPaid *big.Int, isFeesInBase bool) *big.Int {
	maxLiqExpansion := bignum.Two
	bufferTokens := new(big.Int).Mul(maxLiqExpansion, PriceToTokenPrecision(liq, price, isFeesInBase))
	if feesPaid.Cmp(bufferTokens) <= 0 {
		return new(big.Int)
	}
	return new(big.Int).Sub(feesPaid, bufferTokens)
}

func stepToLiquidity(curve *CurveState, inflator uint64, feesInBase bool) {
	curve.PriceRoot = CompoundPrice(curve.PriceRoot, inflator, feesInBase)
	curve.SeedDeflator = CompoundStack(curve.SeedDeflator, inflator)

	concRewards := CompoundShrink(inflator, curve.SeedDeflator)

	newAmbientSeeds := MulQ48(curve.ConcLiq, concRewards)

	curve.ConcGrowth += roundDownConcRewards(concRewards, newAmbientSeeds)
	curve.AmbientSeeds = new(big.Int).Add(curve.AmbientSeeds, newAmbientSeeds)
}

func roundDownConcRewards(concInflator uint64, newAmbientSeeds *big.Int) uint64 {
	if newAmbientSeeds.Sign() <= 0 {
		return 0
	}
	num := new(big.Int).SetUint64(concInflator)
	num.Mul(num, newAmbientSeeds)
	denom := new(big.Int).Add(newAmbientSeeds, bignum.One)
	return num.Div(num, denom).Uint64()
}
