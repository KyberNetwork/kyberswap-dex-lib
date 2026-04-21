package ambient

import (
	"errors"
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var roundPrecisionWei = bignum.Four

type SwapAccum struct {
	BaseFlow  *big.Int
	QuoteFlow *big.Int
	ProtoFees *big.Int

	CrossInitTickLoops int
	PinSpillLoops      int
	KnockoutCrossLoops int
}

func NewSwapAccum() *SwapAccum {
	return &SwapAccum{
		BaseFlow:  new(big.Int),
		QuoteFlow: new(big.Int),
		ProtoFees: new(big.Int),
	}
}

func (a *SwapAccum) Accumulate(baseFlow, quoteFlow, protoFees *big.Int) {
	a.BaseFlow.Add(a.BaseFlow, baseFlow)
	a.QuoteFlow.Add(a.QuoteFlow, quoteFlow)
	a.ProtoFees.Add(a.ProtoFees, protoFees)
}

func RollFlow(curve *CurveState, flow *big.Int, inBaseQty, isBuy bool, swapQty *big.Int) (paidBase, paidQuote, qtyLeft *big.Int) {
	counterFlow, nextPrice := DeriveImpact(curve, flow, inBaseQty, isBuy)
	paidFlow, paidCounter := signFlow(flow, counterFlow, inBaseQty, isBuy)
	return setCurvePos(curve, inBaseQty, isBuy, swapQty, nextPrice, paidFlow, paidCounter)
}

func RollPrice(curve *CurveState, price *big.Int, inBaseQty, isBuy bool, swapQty *big.Int) (paidBase, paidQuote, qtyLeft *big.Int) {
	flow, counterFlow := deriveDemand(curve, price, inBaseQty)
	paidFlow, paidCounter := signFixed(flow, counterFlow, inBaseQty, isBuy)
	return setCurvePos(curve, inBaseQty, isBuy, swapQty, price, paidFlow, paidCounter)
}

func DeriveImpact(curve *CurveState, flow *big.Int, inBaseQty, isBuy bool) (counterFlow, nextPrice *big.Int) {
	liq := ActiveLiquidity(curve)
	nextPrice = deriveFlowPrice(curve.PriceRoot, liq, flow, inBaseQty, isBuy)
	if !inBaseQty {
		counterFlow = DeltaBase(liq, curve.PriceRoot, nextPrice)
	} else {
		counterFlow = DeltaQuote(liq, curve.PriceRoot, nextPrice)
	}
	return
}

func deriveDemand(curve *CurveState, price *big.Int, inBaseQty bool) (flow, counterFlow *big.Int) {
	liq := ActiveLiquidity(curve)
	baseFlow := DeltaBase(liq, curve.PriceRoot, price)
	quoteFlow := DeltaQuote(liq, curve.PriceRoot, price)
	if inBaseQty {
		return baseFlow, quoteFlow
	}
	return quoteFlow, baseFlow
}

func deriveFlowPrice(price, liq, flow *big.Int, inBaseQty, isBuy bool) *big.Int {
	var curvePrice *big.Int
	if inBaseQty {
		curvePrice = calcBaseFlowPrice(price, liq, flow, isBuy)
	} else {
		curvePrice = calcQuoteFlowPrice(price, liq, flow, isBuy)
	}
	if curvePrice.Cmp(MaxSqrtRatio) >= 0 {
		return new(big.Int).Set(MaxSqrtRatioMinus1)
	}
	if curvePrice.Cmp(MinSqrtRatio) < 0 {
		return new(big.Int).Set(MinSqrtRatio)
	}
	return curvePrice
}

func calcBaseFlowPrice(price, liq, flow *big.Int, isBuy bool) *big.Int {
	if liq.Sign() == 0 {
		return new(big.Int).Set(mask128)
	}
	deltaCalc := DivQ64(flow, liq)
	if deltaCalc.Cmp(mask128) > 0 {
		return new(big.Int).Set(mask128)
	}
	if isBuy {
		return new(big.Int).Add(price, deltaCalc)
	}
	if deltaCalc.Cmp(price) >= 0 {
		return new(big.Int)
	}
	result := new(big.Int).Sub(price, deltaCalc)
	result.Sub(result, bignum.One)
	return result
}

func calcQuoteFlowPrice(price, liq, flow *big.Int, isBuy bool) *big.Int {
	invPrice := RecipQ64(price)
	invNext := calcBaseFlowPrice(invPrice, liq, flow, !isBuy)
	if invNext.Sign() == 0 {
		return new(big.Int).Set(MaxSqrtRatio)
	}
	result := RecipQ64(invNext)
	result.Add(result, bignum.One)
	return result
}

func signFlow(flowMagn, counterMagn *big.Int, inBaseQty, isBuy bool) (flow, counter *big.Int) {
	flow, counter = signMagn(flowMagn, counterMagn, inBaseQty, isBuy)
	counter.Add(counter, roundPrecisionWei)
	return
}

func signFixed(flowMagn, counterMagn *big.Int, inBaseQty, isBuy bool) (flow, counter *big.Int) {
	flow, counter = signMagn(flowMagn, counterMagn, inBaseQty, isBuy)
	flow.Add(flow, roundPrecisionWei)
	counter.Add(counter, roundPrecisionWei)
	return
}

func signMagn(flowMagn, counterMagn *big.Int, inBaseQty, isBuy bool) (flow, counter *big.Int) {
	if inBaseQty == isBuy {
		flow = new(big.Int).Set(flowMagn)
		counter = new(big.Int).Neg(counterMagn)
	} else {
		flow = new(big.Int).Neg(flowMagn)
		counter = new(big.Int).Set(counterMagn)
	}
	return
}

func setCurvePos(curve *CurveState, inBaseQty, isBuy bool, swapQty, price, paidFlow, paidCounter *big.Int) (paidBase, paidQuote, qtyLeft *big.Int) {
	spent := flowToSpent(paidFlow, inBaseQty, isBuy)
	if spent.Cmp(swapQty) >= 0 {
		qtyLeft = new(big.Int)
	} else {
		qtyLeft = new(big.Int).Sub(swapQty, spent)
	}
	if inBaseQty {
		paidBase, paidQuote = paidFlow, paidCounter
	} else {
		paidBase, paidQuote = paidCounter, paidFlow
	}
	curve.PriceRoot = new(big.Int).Set(price)
	return
}

func flowToSpent(paidFlow *big.Int, inBaseQty, isBuy bool) *big.Int {
	if inBaseQty == isBuy {
		if paidFlow.Sign() < 0 {
			return new(big.Int)
		}
		return paidFlow
	}
	spent := new(big.Int).Neg(paidFlow)
	if spent.Sign() < 0 {
		return new(big.Int)
	}
	return spent
}

func PriceToTokenPrecision(liq, price *big.Int, inBase bool) *big.Int {
	if inBase {
		result := new(big.Int).Rsh(liq, 64)
		result.Add(result, bignum.One)
		return result
	}
	priceMinus1 := new(big.Int).Sub(price, bignum.One)
	step := DivQ64(liq, priceMinus1)
	start := DivQ64(liq, price)
	delta := new(big.Int).Sub(step, start)
	delta.Add(delta, bignum.One)
	return delta
}

// ErrShaveBurnDown mirrors Solidity's require(swapLeft > burnDown).
// In practice unreachable; kept as error (not panic) for safety.
var ErrShaveBurnDown = errors.New("shave-at-bump: swapLeft <= burnDown")

func ShaveAtBump(curve *CurveState, inBaseQty, isBuy bool, swapLeft *big.Int) (paidBase, paidQuote, burnSwap *big.Int, err error) {
	liq := ActiveLiquidity(curve)
	burnDown := PriceToTokenPrecision(liq, curve.PriceRoot, inBaseQty)
	if swapLeft.Cmp(burnDown) <= 0 {
		return nil, nil, nil, ErrShaveBurnDown
	}
	if isBuy {
		paidBase, paidQuote, burnSwap = setShaveUp(curve, inBaseQty, burnDown)
		return paidBase, paidQuote, burnSwap, nil
	}
	paidBase, paidQuote, burnSwap = setShaveDown(curve, inBaseQty, burnDown)
	return paidBase, paidQuote, burnSwap, nil
}

func setShaveDown(curve *CurveState, inBaseQty bool, burnDown *big.Int) (paidBase, paidQuote, burnSwap *big.Int) {
	if curve.PriceRoot.Cmp(MinSqrtRatio) > 0 {
		curve.PriceRoot = new(big.Int).Sub(curve.PriceRoot, bignum.One)
	}
	paidBase = new(big.Int)
	paidQuote = new(big.Int).Set(burnDown)
	if inBaseQty {
		burnSwap = new(big.Int)
	} else {
		burnSwap = new(big.Int).Set(burnDown)
	}
	return
}

func setShaveUp(curve *CurveState, inBaseQty bool, burnDown *big.Int) (paidBase, paidQuote, burnSwap *big.Int) {
	if curve.PriceRoot.Cmp(MaxSqrtRatioMinus1) < 0 {
		curve.PriceRoot = new(big.Int).Add(curve.PriceRoot, bignum.One)
	}
	paidQuote = new(big.Int)
	paidBase = new(big.Int).Set(burnDown)
	if inBaseQty {
		burnSwap = new(big.Int).Set(burnDown)
	} else {
		burnSwap = new(big.Int)
	}
	return
}
