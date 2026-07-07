package ambient

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// roundPrecisionWei is added to counter-flows to buffer rounding at tick boundaries.
var roundPrecisionWei = uint256.NewInt(4)

type SwapAccum struct {
	BaseFlow  uint256.Int // signed two's complement
	QuoteFlow uint256.Int // signed two's complement
	ProtoFees uint256.Int // signed two's complement

	CrossInitTickLoops int
	PinSpillLoops      int
	KnockoutCrossLoops int
}

func NewSwapAccum() *SwapAccum {
	return &SwapAccum{}
}

func (a *SwapAccum) Accumulate(baseFlow, quoteFlow, protoFees uint256.Int) {
	a.BaseFlow.Add(&a.BaseFlow, &baseFlow)
	a.QuoteFlow.Add(&a.QuoteFlow, &quoteFlow)
	a.ProtoFees.Add(&a.ProtoFees, &protoFees)
}

func RollFlow(curve *CurveState, flow uint256.Int, inBaseQty, isBuy bool, swapQty uint256.Int) (paidBase, paidQuote uint256.Int, qtyLeft uint256.Int) {
	counterFlow, nextPrice := DeriveImpact(curve, flow, inBaseQty, isBuy)
	paidFlow, paidCounter := signFlow(flow, counterFlow, inBaseQty, isBuy)
	return setCurvePos(curve, inBaseQty, isBuy, swapQty, nextPrice, paidFlow, paidCounter)
}

func RollPrice(curve *CurveState, price uint256.Int, inBaseQty, isBuy bool, swapQty uint256.Int) (paidBase, paidQuote uint256.Int, qtyLeft uint256.Int) {
	flow, counterFlow := deriveDemand(curve, price, inBaseQty)
	paidFlow, paidCounter := signFixed(flow, counterFlow, inBaseQty, isBuy)
	return setCurvePos(curve, inBaseQty, isBuy, swapQty, price, paidFlow, paidCounter)
}

func DeriveImpact(curve *CurveState, flow uint256.Int, inBaseQty, isBuy bool) (counterFlow uint256.Int, nextPrice uint256.Int) {
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	nextPrice = deriveFlowPrice(curve.PriceRoot, liq, flow, inBaseQty, isBuy)
	if !inBaseQty {
		DeltaBase(&counterFlow, &liq, &curve.PriceRoot, &nextPrice)
	} else {
		DeltaQuote(&counterFlow, &liq, &curve.PriceRoot, &nextPrice)
	}
	return
}

func deriveDemand(curve *CurveState, price uint256.Int, inBaseQty bool) (flow, counterFlow uint256.Int) {
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	var baseFlow, quoteFlow uint256.Int
	DeltaBase(&baseFlow, &liq, &curve.PriceRoot, &price)
	DeltaQuote(&quoteFlow, &liq, &curve.PriceRoot, &price)
	if inBaseQty {
		return baseFlow, quoteFlow
	}
	return quoteFlow, baseFlow
}

func deriveFlowPrice(price, liq, flow uint256.Int, inBaseQty, isBuy bool) uint256.Int {
	var curvePrice uint256.Int
	if inBaseQty {
		curvePrice = calcBaseFlowPrice(price, liq, flow, isBuy)
	} else {
		curvePrice = calcQuoteFlowPrice(price, liq, flow, isBuy)
	}
	if curvePrice.Cmp(&uMaxSqrtRatio) >= 0 {
		return uMaxSqrtRatioMinus1
	}
	if curvePrice.Lt(&uMinSqrtRatio) {
		return uMinSqrtRatio
	}
	return curvePrice
}

func calcBaseFlowPrice(price, liq, flow uint256.Int, isBuy bool) uint256.Int {
	if liq.IsZero() {
		return uMaxSqrtRatioMinus1
	}
	var deltaCalc uint256.Int
	DivQ64(&deltaCalc, &flow, &liq)
	if deltaCalc.Gt(u256.UMaxU128) {
		return uMaxSqrtRatioMinus1
	}
	if isBuy {
		var result uint256.Int
		result.Add(&price, &deltaCalc)
		return result
	}
	if deltaCalc.Cmp(&price) >= 0 {
		var zero uint256.Int
		return zero
	}
	var result uint256.Int
	result.Sub(&price, &deltaCalc)
	result.Sub(&result, u256.U1)
	return result
}

func calcQuoteFlowPrice(price, liq, flow uint256.Int, isBuy bool) uint256.Int {
	var invPrice uint256.Int
	RecipQ64(&invPrice, &price)
	invNext := calcBaseFlowPrice(invPrice, liq, flow, !isBuy)
	if invNext.IsZero() {
		return uMaxSqrtRatio
	}
	var result uint256.Int
	RecipQ64(&result, &invNext)
	result.Add(&result, u256.U1)
	return result
}

// signFlow applies sign to (flowMagn, counterMagn) based on direction, then adds
// roundPrecisionWei to the counter to buffer rounding.
func signFlow(flowMagn, counterMagn uint256.Int, inBaseQty, isBuy bool) (flow, counter uint256.Int) {
	flow, counter = signMagn(flowMagn, counterMagn, inBaseQty, isBuy)
	counter.Add(&counter, roundPrecisionWei)
	return
}

func signFixed(flowMagn, counterMagn uint256.Int, inBaseQty, isBuy bool) (flow, counter uint256.Int) {
	flow, counter = signMagn(flowMagn, counterMagn, inBaseQty, isBuy)
	flow.Add(&flow, roundPrecisionWei)
	counter.Add(&counter, roundPrecisionWei)
	return
}

// signMagn assigns signs: the "paid" direction is positive, the "received" direction is negative.
func signMagn(flowMagn, counterMagn uint256.Int, inBaseQty, isBuy bool) (flow, counter uint256.Int) {
	if inBaseQty == isBuy {
		// flow is positive (user pays), counter is negative (user receives)
		flow.Set(&flowMagn)
		counter.Neg(&counterMagn)
	} else {
		// flow is negative (user receives), counter is positive (user pays)
		flow.Neg(&flowMagn)
		counter.Set(&counterMagn)
	}
	return
}

func setCurvePos(curve *CurveState, inBaseQty, isBuy bool, swapQty, price uint256.Int, paidFlow, paidCounter uint256.Int) (paidBase, paidQuote uint256.Int, qtyLeft uint256.Int) {
	spent := flowToSpent(paidFlow, inBaseQty, isBuy)
	if spent.Cmp(&swapQty) < 0 {
		qtyLeft.Sub(&swapQty, &spent)
	}
	if inBaseQty {
		paidBase, paidQuote = paidFlow, paidCounter
	} else {
		paidBase, paidQuote = paidCounter, paidFlow
	}
	curve.PriceRoot.Set(&price)
	return
}

func flowToSpent(paidFlow uint256.Int, inBaseQty, isBuy bool) uint256.Int {
	// The "spent" side is whichever flow is positive (user pays).
	if (inBaseQty == isBuy) == (paidFlow.Sign() > 0) {
		var abs uint256.Int
		if paidFlow.Sign() < 0 {
			abs.Neg(&paidFlow)
		} else {
			abs.Set(&paidFlow)
		}
		return abs
	}
	return uint256.Int{}
}

// PriceToTokenPrecision computes the minimum token precision buffer at the current price/liq.
func PriceToTokenPrecision(dst, liq, price *uint256.Int, inBase bool) *uint256.Int {
	if inBase {
		dst.Rsh(liq, 64)
		return dst.Add(dst, u256.U1)
	}
	var priceMinus1, step, start uint256.Int
	priceMinus1.Sub(price, u256.U1)
	DivQ64(&step, liq, &priceMinus1)
	DivQ64(&start, liq, price)
	dst.Sub(&step, &start)
	return dst.Add(dst, u256.U1)
}

var ErrShaveBurnDown = errors.New("shave-at-bump: swapLeft <= burnDown")

func ShaveAtBump(curve *CurveState, inBaseQty, isBuy bool, swapLeft uint256.Int) (paidBase, paidQuote uint256.Int, burnSwap uint256.Int, err error) {
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	var burnDown uint256.Int
	PriceToTokenPrecision(&burnDown, &liq, &curve.PriceRoot, inBaseQty)
	if !swapLeft.Gt(&burnDown) {
		return paidBase, paidQuote, burnSwap, ErrShaveBurnDown
	}
	if isBuy {
		paidBase, paidQuote, burnSwap = setShaveUp(curve, inBaseQty, burnDown)
	} else {
		paidBase, paidQuote, burnSwap = setShaveDown(curve, inBaseQty, burnDown)
	}
	return paidBase, paidQuote, burnSwap, nil
}

func setShaveDown(curve *CurveState, inBaseQty bool, burnDown uint256.Int) (paidBase, paidQuote uint256.Int, burnSwap uint256.Int) {
	if curve.PriceRoot.Gt(&uMinSqrtRatio) {
		curve.PriceRoot.Sub(&curve.PriceRoot, u256.U1)
	}
	// paidBase = 0 (zero value), paidQuote = burnDown
	paidQuote.Set(&burnDown)
	if !inBaseQty {
		burnSwap = burnDown
	}
	return
}

func setShaveUp(curve *CurveState, inBaseQty bool, burnDown uint256.Int) (paidBase, paidQuote uint256.Int, burnSwap uint256.Int) {
	if curve.PriceRoot.Lt(&uMaxSqrtRatioMinus1) {
		curve.PriceRoot.Add(&curve.PriceRoot, u256.U1)
	}
	// paidQuote = 0 (zero value), paidBase = burnDown
	paidBase.Set(&burnDown)
	if inBaseQty {
		burnSwap = burnDown
	}
	return
}

// FlowToBig converts a signed two's-complement uint256 to *big.Int.
// f is passed by value so &f is a safe local pointer for big256.ToBig.
func FlowToBig(f uint256.Int) *big.Int {
	return u256.ToBig(&f)
}
