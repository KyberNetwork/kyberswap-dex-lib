package ambient

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type SwapDirective struct {
	Qty        uint256.Int
	InBaseQty  bool
	IsBuy      bool
	LimitPrice uint256.Int
}

type PoolParams struct {
	FeeRate      uint16
	ProtocolTake uint8
	TickSize     uint16
}

func SwapToLimit(curve *CurveState, accum *SwapAccum, swap *SwapDirective, pool *PoolParams, bumpTick int32) {
	limitPrice := determineLimit(bumpTick, swap.LimitPrice, swap.IsBuy)

	paidBase, paidQuote, paidProto := bookExchFees(curve, &swap.Qty, pool, swap.InBaseQty, limitPrice)
	accum.Accumulate(paidBase, paidQuote, paidProto)

	paidBase, paidQuote, swap.Qty = swapOverCurve(curve, swap.InBaseQty, swap.IsBuy, swap.Qty, limitPrice)
	accum.Accumulate(paidBase, paidQuote, uint256.Int{})
}

func CalcLimitFlows(curve *CurveState, swapQty *uint256.Int, inBaseQty bool, limitPrice uint256.Int) uint256.Int {
	limitFlow := calcLimitFlowsUncapped(curve, inBaseQty, limitPrice)
	if limitFlow.Gt(swapQty) {
		return *swapQty
	}
	return limitFlow
}

func calcLimitFlowsUncapped(curve *CurveState, inBaseQty bool, limitPrice uint256.Int) uint256.Int {
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	var result uint256.Int
	if inBaseQty {
		DeltaBase(&result, &liq, &curve.PriceRoot, &limitPrice)
	} else {
		DeltaQuote(&result, &liq, &curve.PriceRoot, &limitPrice)
	}
	return result
}

func CalcLimitCounter(curve *CurveState, swapQty *uint256.Int, inBaseQty bool, limitPrice uint256.Int) uint256.Int {
	isBuy := limitPrice.Gt(&curve.PriceRoot)
	denomFlow := CalcLimitFlows(curve, swapQty, inBaseQty, limitPrice)
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)
	return invertFlow(liq, curve.PriceRoot, denomFlow, isBuy, inBaseQty)
}

func invertFlow(liq, price, denomFlow uint256.Int, isBuy, inBaseQty bool) uint256.Int {
	if liq.IsZero() {
		var zero uint256.Int
		return zero
	}
	var invertReserve, initReserve uint256.Int
	ReserveAtPrice(&invertReserve, &liq, &price, !inBaseQty)
	ReserveAtPrice(&initReserve, &liq, &price, inBaseQty)

	var endReserve uint256.Int
	if isBuy == inBaseQty {
		endReserve.Add(&initReserve, &denomFlow)
	} else {
		endReserve.Sub(&initReserve, &denomFlow)
	}
	if endReserve.IsZero() {
		return *u256.UMaxU128
	}

	var liqSq uint256.Int
	liqSq.Mul(&liq, &liq)
	var endInvert uint256.Int
	endInvert.Div(&liqSq, &endReserve)

	var diff uint256.Int
	if endInvert.Gt(&invertReserve) {
		diff.Sub(&endInvert, &invertReserve)
	} else {
		diff.Sub(&invertReserve, &endInvert)
	}
	return diff
}

func swapOverCurve(curve *CurveState, inBaseQty, isBuy bool, swapQty, limitPrice uint256.Int) (paidBase, paidQuote uint256.Int, qtyLeft uint256.Int) {
	realFlows := CalcLimitFlows(curve, &swapQty, inBaseQty, limitPrice)
	hitsLimit := realFlows.Lt(&swapQty)

	if hitsLimit {
		return RollPrice(curve, limitPrice, inBaseQty, isBuy, swapQty)
	}
	return RollFlow(curve, realFlows, inBaseQty, isBuy, swapQty)
}

func determineLimit(bumpTick int32, limitPrice uint256.Int, isBuy bool) uint256.Int {
	bounded := boundLimit(bumpTick, limitPrice, isBuy)
	if bounded.Lt(&uMinSqrtRatio) {
		return uMinSqrtRatio
	}
	if bounded.Cmp(&uMaxSqrtRatio) >= 0 {
		return uMaxSqrtRatioMinus1
	}
	return bounded
}

func boundLimit(bumpTick int32, limitPrice uint256.Int, isBuy bool) uint256.Int {
	if bumpTick <= MinTick || bumpTick >= MaxTick {
		return limitPrice
	}
	if isBuy {
		bumpPrice := GetSqrtRatioAtTick(bumpTick)
		bumpPrice.Sub(&bumpPrice, u256.U1)
		if bumpPrice.Lt(&limitPrice) {
			return bumpPrice
		}
		return limitPrice
	}
	bumpPrice := GetSqrtRatioAtTick(bumpTick)
	if bumpPrice.Gt(&limitPrice) {
		return bumpPrice
	}
	return limitPrice
}

func bookExchFees(curve *CurveState, swapQty *uint256.Int, pool *PoolParams, inBaseQty bool, limitPrice uint256.Int) (paidBase, paidQuote, paidProto uint256.Int) {
	flow := CalcLimitCounter(curve, swapQty, inBaseQty, limitPrice)
	liqFees, exchFees := CalcFeeOverFlow(flow, pool.FeeRate, pool.ProtocolTake)

	AssimilateLiq(curve, &liqFees, inBaseQty)

	return assignFees(liqFees, exchFees, inBaseQty)
}

func assignFees(liqFees, exchFees uint256.Int, inBaseQty bool) (paidBase, paidQuote, paidProto uint256.Int) {
	var totalFees uint256.Int
	totalFees.Add(&liqFees, &exchFees)
	if inBaseQty {
		paidQuote = totalFees
	} else {
		paidBase = totalFees
	}
	paidProto = exchFees
	return
}

var (
	uFeeBPMult        = *uint256.NewInt(1_000_000)
	uProtoTakeDivisor = *uint256.NewInt(256)
)

func CalcFeeOverFlow(flow uint256.Int, feeRate uint16, protoProp uint8) (liqFee, protoFee uint256.Int) {
	liqFee.SetUint64(uint64(feeRate))
	liqFee.Mul(&flow, &liqFee)
	liqFee.Div(&liqFee, &uFeeBPMult)
	protoFee.SetUint64(uint64(protoProp))
	protoFee.Mul(&liqFee, &protoFee)
	protoFee.Div(&protoFee, &uProtoTakeDivisor)
	liqFee.Sub(&liqFee, &protoFee)
	return
}
