package ambient

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func AssimilateLiq(curve *CurveState, feesPaid *uint256.Int, isSwapInBase bool) {
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	if liq.IsZero() {
		return
	}
	feesInBase := !isSwapInBase

	var feesToLiq uint256.Int
	shaveForPrecision(&feesToLiq, &liq, &curve.PriceRoot, feesPaid, feesInBase)
	if feesToLiq.IsZero() {
		return
	}
	inflator := calcLiqInflator(&liq, &curve.PriceRoot, &feesToLiq, feesInBase)
	if inflator > 0 {
		stepToLiquidity(curve, inflator, feesInBase)
	}
}

func calcLiqInflator(liq, price, feesPaid *uint256.Int, inBaseQty bool) uint64 {
	var reserve uint256.Int
	ReserveAtPrice(&reserve, liq, price, inBaseQty)
	return calcReserveInflator(&reserve, feesPaid)
}

func calcReserveInflator(reserve, feesPaid *uint256.Int) uint64 {
	if reserve.IsZero() || feesPaid.Gt(reserve) {
		return 0
	}
	var nextReserve uint256.Int
	nextReserve.Add(reserve, feesPaid)
	inflatorRoot := CompoundDivide(&nextReserve, reserve)
	return ApproxSqrtCompound(inflatorRoot)
}

func shaveForPrecision(dst, liq, price, feesPaid *uint256.Int, isFeesInBase bool) *uint256.Int {
	var precision uint256.Int
	PriceToTokenPrecision(&precision, liq, price, isFeesInBase)
	// bufferTokens = 2 * precision
	precision.Lsh(&precision, 1)
	if !feesPaid.Gt(&precision) {
		dst.Clear()
		return dst
	}
	dst.Sub(feesPaid, &precision)
	return dst
}

func stepToLiquidity(curve *CurveState, inflator uint64, feesInBase bool) {
	CompoundPrice(&curve.PriceRoot, &curve.PriceRoot, inflator, feesInBase)
	curve.SeedDeflator = CompoundStack(curve.SeedDeflator, inflator)

	concRewards := CompoundShrink(inflator, curve.SeedDeflator)

	var newAmbientSeeds uint256.Int
	MulQ48(&newAmbientSeeds, &curve.ConcLiq, concRewards)

	curve.ConcGrowth += roundDownConcRewards(concRewards, &newAmbientSeeds)
	curve.AmbientSeeds.Add(&curve.AmbientSeeds, &newAmbientSeeds)
}

func roundDownConcRewards(concInflator uint64, newAmbientSeeds *uint256.Int) uint64 {
	if newAmbientSeeds.IsZero() {
		return 0
	}
	var num, denom uint256.Int
	num.SetUint64(concInflator)
	num.Mul(&num, newAmbientSeeds)
	denom.Add(newAmbientSeeds, u256.U1)
	return num.Div(&num, &denom).Uint64()
}
