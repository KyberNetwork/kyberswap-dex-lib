package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type SwapDirective struct {
	Qty        *big.Int
	InBaseQty  bool
	IsBuy      bool
	LimitPrice *big.Int
}

type PoolParams struct {
	FeeRate      uint16
	ProtocolTake uint8
	TickSize     uint16
}

func SwapToLimit(curve *CurveState, accum *SwapAccum, swap *SwapDirective, pool *PoolParams, bumpTick int32) {
	limitPrice := determineLimit(bumpTick, swap.LimitPrice, swap.IsBuy)

	paidBase, paidQuote, paidProto := bookExchFees(curve, swap.Qty, pool, swap.InBaseQty, limitPrice)
	accum.Accumulate(paidBase, paidQuote, paidProto)

	paidBase, paidQuote, swap.Qty = swapOverCurve(curve, swap.InBaseQty, swap.IsBuy, swap.Qty, limitPrice)
	accum.Accumulate(paidBase, paidQuote, bignum.ZeroBI)
}

func CalcLimitFlows(curve *CurveState, swapQty *big.Int, inBaseQty bool, limitPrice *big.Int) *big.Int {
	limitFlow := calcLimitFlowsUncapped(curve, inBaseQty, limitPrice)
	if limitFlow.Cmp(swapQty) > 0 {
		return new(big.Int).Set(swapQty)
	}
	return limitFlow
}

func calcLimitFlowsUncapped(curve *CurveState, inBaseQty bool, limitPrice *big.Int) *big.Int {
	liq := ActiveLiquidity(curve)
	if inBaseQty {
		return DeltaBase(liq, curve.PriceRoot, limitPrice)
	}
	return DeltaQuote(liq, curve.PriceRoot, limitPrice)
}

func CalcLimitCounter(curve *CurveState, swapQty *big.Int, inBaseQty bool, limitPrice *big.Int) *big.Int {
	isBuy := limitPrice.Cmp(curve.PriceRoot) > 0
	denomFlow := CalcLimitFlows(curve, swapQty, inBaseQty, limitPrice)
	return invertFlow(ActiveLiquidity(curve), curve.PriceRoot, denomFlow, isBuy, inBaseQty)
}

func invertFlow(liq, price, denomFlow *big.Int, isBuy, inBaseQty bool) *big.Int {
	if liq.Sign() == 0 {
		return new(big.Int)
	}
	invertReserve := ReserveAtPrice(liq, price, !inBaseQty)
	initReserve := ReserveAtPrice(liq, price, inBaseQty)

	var endReserve *big.Int
	if isBuy == inBaseQty {
		endReserve = new(big.Int).Add(initReserve, denomFlow)
	} else {
		endReserve = new(big.Int).Sub(initReserve, denomFlow)
	}
	if endReserve.Sign() == 0 {
		return new(big.Int).Set(mask128)
	}

	liqSq := new(big.Int).Mul(liq, liq)
	endInvert := new(big.Int).Div(liqSq, endReserve)

	diff := new(big.Int).Sub(endInvert, invertReserve)
	if diff.Sign() < 0 {
		diff.Neg(diff)
	}
	return diff
}

func swapOverCurve(curve *CurveState, inBaseQty, isBuy bool, swapQty, limitPrice *big.Int) (paidBase, paidQuote, qtyLeft *big.Int) {
	realFlows := CalcLimitFlows(curve, swapQty, inBaseQty, limitPrice)
	hitsLimit := realFlows.Cmp(swapQty) < 0

	if hitsLimit {
		return RollPrice(curve, limitPrice, inBaseQty, isBuy, swapQty)
	}
	return RollFlow(curve, realFlows, inBaseQty, isBuy, swapQty)
}

func determineLimit(bumpTick int32, limitPrice *big.Int, isBuy bool) *big.Int {
	bounded := boundLimit(bumpTick, limitPrice, isBuy)
	if bounded.Cmp(MinSqrtRatio) < 0 {
		return new(big.Int).Set(MinSqrtRatio)
	}
	if bounded.Cmp(MaxSqrtRatio) >= 0 {
		return new(big.Int).Set(MaxSqrtRatioMinus1)
	}
	return bounded
}

func boundLimit(bumpTick int32, limitPrice *big.Int, isBuy bool) *big.Int {
	if bumpTick <= MinTick || bumpTick >= MaxTick {
		return new(big.Int).Set(limitPrice)
	}
	if isBuy {
		bumpPrice := new(big.Int).Sub(GetSqrtRatioAtTick(bumpTick), bignum.One)
		if bumpPrice.Cmp(limitPrice) < 0 {
			return bumpPrice
		}
		return new(big.Int).Set(limitPrice)
	}
	bumpPrice := GetSqrtRatioAtTick(bumpTick)
	if bumpPrice.Cmp(limitPrice) > 0 {
		return bumpPrice
	}
	return new(big.Int).Set(limitPrice)
}

func bookExchFees(curve *CurveState, swapQty *big.Int, pool *PoolParams, inBaseQty bool, limitPrice *big.Int) (paidBase, paidQuote, paidProto *big.Int) {
	flow := CalcLimitCounter(curve, swapQty, inBaseQty, limitPrice)
	liqFees, exchFees := CalcFeeOverFlow(flow, pool.FeeRate, pool.ProtocolTake)

	AssimilateLiq(curve, liqFees, inBaseQty)

	return assignFees(liqFees, exchFees, inBaseQty)
}

func assignFees(liqFees, exchFees *big.Int, inBaseQty bool) (paidBase, paidQuote, paidProto *big.Int) {
	totalFees := liqFees.Add(liqFees, exchFees)
	paidBase = bignum.ZeroBI
	paidQuote = bignum.ZeroBI
	if inBaseQty {
		paidQuote = totalFees
	} else {
		paidBase = totalFees
	}
	paidProto = exchFees
	return
}

var (
	feeBPMult        = big.NewInt(1_000_000)
	protoTakeDivisor = big.NewInt(256)
)

func CalcFeeOverFlow(flow *big.Int, feeRate uint16, protoProp uint8) (liqFee, protoFee *big.Int) {
	totalFee := new(big.Int).SetUint64(uint64(feeRate))
	totalFee.Mul(flow, totalFee)
	totalFee.Div(totalFee, feeBPMult)
	protoFee = new(big.Int).SetUint64(uint64(protoProp))
	protoFee.Mul(totalFee, protoFee)
	protoFee.Div(protoFee, protoTakeDivisor)
	liqFee = new(big.Int).Sub(totalFee, protoFee)
	return
}
