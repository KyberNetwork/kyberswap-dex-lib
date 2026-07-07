package baseline

import (
	"math/big"

	"github.com/holiman/uint256"
)

func (p *PoolSimulator) quoteAmountOut(isBuy bool, amountIn *big.Int) (*quoteResult, error) {
	if p.extra.QuoteState == nil || p.extra.QuoteState.SnapshotCurveParams.BLV == nil {
		return nil, ErrNoRate
	}

	state := cloneQuoteState(p.extra.QuoteState)
	if isBuy {
		return quoteBuyExactIn(state, amountIn)
	}
	return quoteSellExactIn(state, amountIn)
}

func (p *PoolSimulator) quoteAmountIn(isBuy bool, amountOut *big.Int) (*quoteResult, error) {
	if p.extra.QuoteState == nil || p.extra.QuoteState.SnapshotCurveParams.BLV == nil {
		return nil, ErrNoRate
	}

	state := cloneQuoteState(p.extra.QuoteState)
	if isBuy {
		return quoteBuyExactOut(state, amountOut)
	}
	return quoteSellExactOut(state, amountOut)
}

func quoteBuyExactIn(state *QuoteState, reservesIn *big.Int) (*quoteResult, error) {
	if reservesIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	tokensOut, _, accountingFee, reserveDelta, err := solveBuy(state, reservesIn)
	if err != nil {
		return nil, err
	}
	if tokensOut.Sign() <= 0 {
		return nil, ErrNoRate
	}

	next := cloneQuoteState(state)
	applyQuoteState(next, tokensOut, reserveDelta, accountingFee)

	return &quoteResult{
		AmountOut:     biToU(tokensOut),
		Fee:           biToU(accountingFee),
		AccountingFee: biToU(accountingFee),
		ReserveDelta:  reserveDelta,
		State:         next,
	}, nil
}

func quoteSellExactIn(state *QuoteState, tokensIn *big.Int) (*quoteResult, error) {
	if tokensIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	deltaCirc := new(big.Int).Neg(tokensIn)
	reserveDelta, fee, err := quoteSwap(state, deltaCirc)
	if err != nil {
		return nil, err
	}
	if reserveDelta.Sign() <= 0 {
		return nil, ErrNoRate
	}

	next := cloneQuoteState(state)
	applyQuoteState(next, deltaCirc, reserveDelta, fee)

	return &quoteResult{
		AmountOut:     biToU(reserveDelta),
		Fee:           biToU(fee),
		AccountingFee: biToU(fee),
		ReserveDelta:  reserveDelta,
		State:         next,
	}, nil
}

func quoteBuyExactOut(state *QuoteState, tokensOut *big.Int) (*quoteResult, error) {
	if tokensOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	cost, fee, err := quoteBuyExactOutCost(state, tokensOut)
	if err != nil {
		return nil, err
	}
	if cost.Sign() <= 0 {
		return nil, ErrNoRate
	}

	reserveDelta := new(big.Int).Neg(cost)
	next := cloneQuoteState(state)
	applyQuoteState(next, tokensOut, reserveDelta, fee)

	return &quoteResult{
		AmountOut:     biToU(cost),
		Fee:           biToU(fee),
		AccountingFee: biToU(fee),
		ReserveDelta:  reserveDelta,
		State:         next,
	}, nil
}

func quoteSellExactOut(state *QuoteState, reservesOut *big.Int) (*quoteResult, error) {
	if reservesOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	tokensIn, publicFee, err := solveSellExactOut(state, reservesOut)
	if err != nil {
		return nil, err
	}
	if tokensIn.Sign() <= 0 {
		return nil, ErrNoRate
	}

	accountingQuote, err := quoteSellExactIn(cloneQuoteState(state), tokensIn)
	if err != nil {
		return nil, err
	}

	next := cloneQuoteState(state)
	deltaCirc := new(big.Int).Neg(tokensIn)
	applyQuoteState(next, deltaCirc, accountingQuote.ReserveDelta, accountingQuote.AccountingFee.ToBig())

	return &quoteResult{
		AmountOut:     biToU(tokensIn),
		Fee:           biToU(publicFee),
		AccountingFee: cloneU(accountingQuote.AccountingFee),
		ReserveDelta:  accountingQuote.ReserveDelta,
		State:         next,
	}, nil
}

func solveBuy(state *QuoteState, target *big.Int) (delta, fee, accountingFee, reserveDelta *big.Int, err error) {
	p := state.SnapshotCurveParams
	price, err := computeActivePrice(p)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	priceWithFee := mulWad(price, addBI(wadBI, mulBI(uToBI(p.SwapFee), twoBI)))
	if priceWithFee.Sign() == 0 {
		return nil, nil, nil, nil, errInvalidCurveState
	}

	estimatedDeltaWad := mulBI(divWad(normalizeWadBI(target, state.ReserveDecimals), priceWithFee), twoBI)
	estimatedDelta := denormalizeWadBI(estimatedDeltaWad, bTokenDecimals)
	maxDelta := divBI(mulBI(uToBI(state.TotalBTokens), big.NewInt(99)), big.NewInt(100))
	if maxDelta == nil || maxDelta.Sign() <= 0 {
		return nil, nil, nil, nil, errSolverFailed
	}

	hi := estimatedDelta
	if hi.Cmp(big.NewInt(2)) < 0 {
		hi = big.NewInt(2)
	}
	if hi.Cmp(maxDelta) > 0 {
		hi = new(big.Int).Set(maxDelta)
	}

	for hi.Cmp(maxDelta) < 0 {
		cost, _, quoteErr := quoteBuyExactOutCost(state, hi)
		if quoteErr == nil && cost.Cmp(target) <= 0 {
			hi.Mul(hi, twoBI)
			if hi.Cmp(maxDelta) > 0 {
				hi.Set(maxDelta)
			}
			continue
		}
		break
	}

	lo := big.NewInt(1)
	delta = new(big.Int).Set(lo)
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := divBI(addBI(lo, hi), twoBI)
		cost, _, quoteErr := quoteBuyExactOutCost(state, mid)
		if quoteErr == nil && cost.Cmp(target) <= 0 {
			lo = mid
			delta.Set(mid)
		} else {
			hi = mid
		}
	}

	cost, quoteFee, err := quoteBuyExactOutCost(state, delta)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if cost.Sign() == 0 || cost.Cmp(target) > 0 {
		return nil, nil, nil, nil, errSolverFailed
	}
	fee = addBI(quoteFee, subBI(target, cost))
	return delta, fee, quoteFee, new(big.Int).Neg(cost), nil
}

func quoteBuyExactOutCost(state *QuoteState, tokensOut *big.Int) (cost, fee *big.Int, err error) {
	reserveDelta, fee, err := quoteSwap(state, tokensOut)
	if err != nil {
		return nil, nil, err
	}
	return absBI(reserveDelta), fee, nil
}

func solveSellExactOut(state *QuoteState, targetReservesOut *big.Int) (tokensIn, fee *big.Int, err error) {
	p := state.SnapshotCurveParams
	price, err := computeActivePrice(p)
	if err != nil {
		return nil, nil, err
	}

	priceWithFee := mulWad(price, subBI(wadBI, mulBI(uToBI(p.SwapFee), twoBI)))
	if priceWithFee.Sign() == 0 {
		return nil, nil, errInvalidCurveState
	}

	estimatedDeltaWad := mulBI(divWad(normalizeWadBI(targetReservesOut, state.ReserveDecimals), priceWithFee), twoBI)
	estimatedDelta := denormalizeWadUpBI(estimatedDeltaWad, bTokenDecimals)
	liveMaxDelta := divBI(mulBI(subBI(uToBI(state.TotalSupply), uToBI(state.TotalBTokens)), big.NewInt(99)), big.NewInt(100))
	maxDelta := uToBI(state.MaxSellDelta)
	if liveMaxDelta.Cmp(maxDelta) < 0 {
		maxDelta = liveMaxDelta
	}
	if maxDelta.Sign() <= 0 {
		return nil, nil, errSolverFailed
	}

	hi := estimatedDelta
	if hi.Cmp(big.NewInt(2)) < 0 {
		hi = big.NewInt(2)
	}
	if hi.Cmp(maxDelta) > 0 {
		hi = new(big.Int).Set(maxDelta)
	}

	for hi.Cmp(maxDelta) < 0 {
		output := quoteSellOutput(state, hi)
		if output.Cmp(targetReservesOut) < 0 {
			hi.Mul(hi, twoBI)
			if hi.Cmp(maxDelta) > 0 {
				hi.Set(maxDelta)
			}
			continue
		}
		break
	}

	if quoteSellOutput(state, hi).Cmp(targetReservesOut) < 0 {
		return nil, nil, errSolverFailed
	}

	lo := big.NewInt(1)
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := divBI(addBI(lo, hi), twoBI)
		midOutput := quoteSellOutput(state, mid)
		if midOutput.Cmp(targetReservesOut) >= 0 {
			hi = mid
		} else {
			lo = mid
		}
	}

	finalQuote, err := quoteSellExactIn(cloneQuoteState(state), hi)
	if err != nil {
		return nil, nil, err
	}
	if finalQuote.AmountOut.ToBig().Cmp(targetReservesOut) < 0 {
		return nil, nil, errSolverFailed
	}
	return hi, addBI(finalQuote.Fee.ToBig(), subBI(finalQuote.AmountOut.ToBig(), targetReservesOut)), nil
}

func quoteSellOutput(state *QuoteState, deltaCirc *big.Int) *big.Int {
	quote, err := quoteSellExactIn(cloneQuoteState(state), deltaCirc)
	if err != nil {
		return new(big.Int)
	}
	return quote.AmountOut.ToBig()
}

func quoteSwap(state *QuoteState, deltaCircNative *big.Int) (deltaUserReservesNative, feesNative *big.Int, err error) {
	if deltaCircNative.Sign() < 0 {
		maxSell := maxSellDelta(state)
		if absBI(deltaCircNative).Cmp(maxSell) > 0 {
			return nil, nil, errTradeExceedsLimit
		}
	}

	var lower, upper *big.Int
	if deltaCircNative.Sign() > 0 {
		lower = uToBI(state.QuoteBlockBuyDeltaCirc)
		upper = addBI(lower, deltaCircNative)
	} else {
		lower = new(big.Int).Neg(uToBI(state.QuoteBlockSellDeltaCirc))
		upper = new(big.Int).Neg(addBI(uToBI(state.QuoteBlockSellDeltaCirc), absBI(deltaCircNative)))
	}

	beforeUser, beforeInv, err := quoteCumulativeFromSnapshot(state, lower)
	if err != nil {
		return nil, nil, err
	}
	afterUser, afterInv, err := quoteCumulativeFromSnapshot(state, upper)
	if err != nil {
		return nil, nil, err
	}

	deltaUserWad := subBI(afterUser, beforeUser)
	deltaInvariantWad := subBI(afterInv, beforeInv)
	if deltaUserWad.Cmp(deltaInvariantWad) > 0 {
		deltaUserWad = deltaInvariantWad
	}

	if deltaUserWad.Sign() < 0 {
		userPay := denormalizeWadUpBI(absBI(deltaUserWad), state.ReserveDecimals)
		curveNeed := denormalizeWadUpBI(absBI(deltaInvariantWad), state.ReserveDecimals)
		return new(big.Int).Neg(userPay), subBI(userPay, curveNeed), nil
	}

	userReceive := denormalizeWadBI(deltaUserWad, state.ReserveDecimals)
	curveRelease := denormalizeWadBI(deltaInvariantWad, state.ReserveDecimals)
	return userReceive, subBI(curveRelease, userReceive), nil
}

func quoteCumulativeFromSnapshot(state *QuoteState, cumulativeDeltaCircNative *big.Int) (userDeltaWad, invariantDeltaWad *big.Int, err error) {
	if cumulativeDeltaCircNative.Sign() == 0 {
		return new(big.Int), new(big.Int), nil
	}
	deltaCircWad := toWadSignedBI(cumulativeDeltaCircNative, bTokenDecimals)
	userDeltaWad, _, invariantDeltaWad, err = computeSwap(state.SnapshotCurveParams, deltaCircWad)
	return userDeltaWad, invariantDeltaWad, err
}

func computeSwap(params CurveParams, deltaCirc *big.Int) (userDelta, fee, invariantDelta *big.Int, err error) {
	circ := uToBI(params.Circ)
	if circ.Sign() == 0 {
		return computeZeroCircSwap(params, deltaCirc)
	}

	c1 := addBI(circ, deltaCirc)
	if c1.Sign() < 0 {
		return nil, nil, nil, errTradeExceedsLimit
	}
	if c1.Sign() == 0 {
		blvValue := mulWad(uToBI(params.BLV), circ)
		receipt := mulWad(blvValue, subBI(wadBI, mulBI(uToBI(params.SwapFee), twoBI)))
		return receipt, subBI(uToBI(params.Reserves), receipt), uToBI(params.Reserves), nil
	}

	x1 := subBI(uToBI(params.Supply), deltaCirc)
	if x1.Sign() <= 0 {
		return nil, nil, nil, errTradeExceedsLimit
	}

	var newBuffer *big.Int
	if x1.Cmp(c1) >= 0 {
		ratio := divWad(x1, c1)
		if err := checkPowLimit(ratio, uToBI(params.ConvexityExp)); err != nil {
			return nil, nil, nil, err
		}
		ratioPowN, err := powWadBI(ratio, uToBI(params.ConvexityExp))
		if err != nil {
			return nil, nil, nil, err
		}
		newBuffer = fullMulDivUp(uToBI(params.LastInvariant), wadBI, ratioPowN)
	} else {
		var invRatio *big.Int
		if deltaCirc.Sign() < 0 {
			invRatio = divWad(c1, x1)
		} else {
			invRatio = divWadUp(c1, x1)
		}
		if err := checkPowLimit(invRatio, uToBI(params.ConvexityExp)); err != nil {
			return nil, nil, nil, err
		}
		invRatioPowN, err := powWadBI(invRatio, uToBI(params.ConvexityExp))
		if err != nil {
			return nil, nil, nil, err
		}
		newBuffer = fullMulDivUp(uToBI(params.LastInvariant), invRatioPowN, wadBI)
	}

	priceBefore, err := computeActivePrice(params)
	if err != nil {
		return nil, nil, nil, err
	}
	priceAfterDenominator := mulWad(x1, c1)
	if priceAfterDenominator.Sign() == 0 {
		return nil, nil, nil, errInvalidCurveState
	}
	priceAfter := addBI(uToBI(params.BLV), fullMulDiv(
		newBuffer,
		mulWad(uToBI(params.ConvexityExp), uToBI(params.TotalSupply)),
		priceAfterDenominator,
	))
	if priceAfter.Cmp(priceBefore) == 0 {
		return nil, nil, nil, errPriceMustChange
	}

	newReserves := addBI(newBuffer, mulWadUp(uToBI(params.BLV), c1))
	invariantDelta = subBI(uToBI(params.Reserves), newReserves)
	fee = computeFee(params, deltaCirc, newBuffer, invariantDelta)
	userDelta = subBI(invariantDelta, fee)
	return userDelta, fee, invariantDelta, nil
}

func computeZeroCircSwap(params CurveParams, deltaCirc *big.Int) (userDelta, fee, invariantDelta *big.Int, err error) {
	if deltaCirc.Sign() <= 0 {
		return nil, nil, nil, errTradeExceedsLimit
	}

	x1 := subBI(uToBI(params.Supply), deltaCirc)
	if x1.Sign() <= 0 {
		return nil, nil, nil, errTradeExceedsLimit
	}

	var newBuffer *big.Int
	if deltaCirc.Cmp(x1) >= 0 {
		ratio := divWadUp(deltaCirc, x1)
		if err := checkPowLimit(ratio, uToBI(params.ConvexityExp)); err != nil {
			return nil, nil, nil, err
		}
		ratioPowN, err := powWadBI(ratio, uToBI(params.ConvexityExp))
		if err != nil {
			return nil, nil, nil, err
		}
		newBuffer = fullMulDivUp(uToBI(params.LastInvariant), ratioPowN, wadBI)
	} else {
		invRatio := divWad(x1, deltaCirc)
		if err := checkPowLimit(invRatio, uToBI(params.ConvexityExp)); err != nil {
			return nil, nil, nil, err
		}
		invRatioPowN, err := powWadBI(invRatio, uToBI(params.ConvexityExp))
		if err != nil {
			return nil, nil, nil, err
		}
		newBuffer = fullMulDivUp(uToBI(params.LastInvariant), wadBI, invRatioPowN)
	}

	invariantDelta = subBI(uToBI(params.Reserves), addBI(newBuffer, mulWadUp(uToBI(params.BLV), deltaCirc)))
	bufferReservesDenominator := mulWad(deltaCirc, x1)
	if bufferReservesDenominator.Sign() == 0 {
		return nil, nil, nil, errInvalidCurveState
	}
	bufferReserves := fullMulDivUp(
		newBuffer,
		mulWadUp(uToBI(params.ConvexityExp), uToBI(params.TotalSupply)),
		bufferReservesDenominator,
	)
	payment := mulWadUp(deltaCirc, addBI(mulWadUp(uToBI(params.BLV), addBI(wadBI, mulBI(uToBI(params.SwapFee), twoBI))), bufferReserves))
	return new(big.Int).Neg(payment), addBI(invariantDelta, payment), invariantDelta, nil
}

func computeFee(params CurveParams, deltaCirc, newBuffer, invariantDelta *big.Int) *big.Int {
	absDelta := absBI(deltaCirc)
	c1 := addBI(uToBI(params.Circ), deltaCirc)
	x1 := subBI(uToBI(params.Supply), deltaCirc)

	if deltaCirc.Sign() > 0 {
		marginalPremium := fullMulDivUp(
			newBuffer,
			mulWadUp(uToBI(params.ConvexityExp), uToBI(params.TotalSupply)),
			mulWad(c1, x1),
		)
		marginalCost := mulWadUp(absDelta, addBI(
			mulWadUp(uToBI(params.BLV), addBI(wadBI, mulBI(uToBI(params.SwapFee), twoBI))),
			marginalPremium,
		))
		return zeroFloorSubBI(marginalCost, absBI(invariantDelta))
	}

	marginalPremium := fullMulDiv(
		newBuffer,
		mulWad(uToBI(params.ConvexityExp), uToBI(params.TotalSupply)),
		mulWadUp(c1, x1),
	)
	marginalReceipt := mulWad(absDelta, addBI(
		uToBI(params.BLV),
		mulWad(marginalPremium, subBI(wadBI, mulBI(uToBI(params.SwapFee), twoBI))),
	))
	return zeroFloorSubBI(invariantDelta, marginalReceipt)
}

func computeActivePrice(params CurveParams) (*big.Int, error) {
	if uToBI(params.Circ).Sign() == 0 {
		return uToBI(params.BLV), nil
	}
	buffer := subBI(uToBI(params.Reserves), mulWad(uToBI(params.BLV), uToBI(params.Circ)))
	if buffer.Sign() < 0 {
		return nil, errInvalidCurveState
	}
	premiumDenominator := mulWad(uToBI(params.Supply), uToBI(params.Circ))
	if premiumDenominator.Sign() == 0 {
		return nil, errInvalidCurveState
	}
	premium := fullMulDiv(
		buffer,
		mulWad(uToBI(params.ConvexityExp), uToBI(params.TotalSupply)),
		premiumDenominator,
	)
	return addBI(uToBI(params.BLV), premium), nil
}

func maxSellDelta(state *QuoteState) *big.Int {
	return uToBI(state.MaxSellDelta)
}

func applyQuoteState(state *QuoteState, deltaCirc, reserveDelta, fee *big.Int) {
	if state == nil {
		return
	}

	settlePendingSurplus(state)

	nextTotalBTokens := subBI(uToBI(state.TotalBTokens), deltaCirc)
	if state.TotalBTokens != nil {
		state.TotalBTokens = biToU(nextTotalBTokens)
	}
	if state.TotalReserves != nil {
		state.TotalReserves = biToU(subBI(subBI(uToBI(state.TotalReserves), reserveDelta), fee))
	}
	recordPendingLiquidityFee(state, nextTotalBTokens, fee)

	if deltaCirc.Sign() > 0 {
		state.QuoteBlockBuyDeltaCirc = biToU(addBI(uToBI(state.QuoteBlockBuyDeltaCirc), deltaCirc))
	} else if deltaCirc.Sign() < 0 {
		state.QuoteBlockSellDeltaCirc = biToU(addBI(uToBI(state.QuoteBlockSellDeltaCirc), absBI(deltaCirc)))
	}
	if state.MaxSellDelta != nil {
		state.MaxSellDelta = biToU(subBI(uToBI(state.MaxSellDelta), absBI(deltaCirc)))
	}
}

func settlePendingSurplus(state *QuoteState) {
	if !state.SettlePendingSurplus || state.TotalReserves == nil || state.PendingSurplus == nil || state.PendingSurplus.IsZero() {
		state.SettlePendingSurplus = false
		return
	}
	bufferThreshold := mulWad(uToBI(state.TotalSupply), mustBI("950000000000000000"))
	if uToBI(state.TotalBTokens).Cmp(bufferThreshold) < 0 {
		state.TotalReserves = biToU(addBI(uToBI(state.TotalReserves), uToBI(state.PendingSurplus)))
	}
	state.PendingSurplus = uint256.NewInt(0)
	state.SettlePendingSurplus = false
}

func recordPendingLiquidityFee(state *QuoteState, nextTotalBTokens, fee *big.Int) {
	if fee.Sign() <= 0 || state.PendingSurplus == nil {
		return
	}
	bufferThreshold := mulWad(uToBI(state.TotalSupply), mustBI("950000000000000000"))
	if nextTotalBTokens.Cmp(bufferThreshold) >= 0 {
		return
	}
	liquidityFee := mulWad(fee, uToBI(state.LiquidityFeePct))
	if liquidityFee.Sign() > 0 {
		state.PendingSurplus = biToU(addBI(uToBI(state.PendingSurplus), liquidityFee))
	}
}

func cloneQuoteState(state *QuoteState) *QuoteState {
	if state == nil {
		return nil
	}
	cloned := *state
	cloned.TotalSupply = cloneU(state.TotalSupply)
	cloned.TotalBTokens = cloneU(state.TotalBTokens)
	cloned.TotalReserves = cloneU(state.TotalReserves)
	cloned.QuoteBlockBuyDeltaCirc = cloneU(state.QuoteBlockBuyDeltaCirc)
	cloned.QuoteBlockSellDeltaCirc = cloneU(state.QuoteBlockSellDeltaCirc)
	cloned.LiquidityFeePct = cloneU(state.LiquidityFeePct)
	cloned.PendingSurplus = cloneU(state.PendingSurplus)
	cloned.MaxSellDelta = cloneU(state.MaxSellDelta)
	cloned.SnapshotActivePrice = cloneU(state.SnapshotActivePrice)
	cloned.SnapshotCurveParams = cloneCurveParams(state.SnapshotCurveParams)
	return &cloned
}

func cloneCurveParams(params CurveParams) CurveParams {
	params.BLV = cloneU(params.BLV)
	params.Circ = cloneU(params.Circ)
	params.Supply = cloneU(params.Supply)
	params.SwapFee = cloneU(params.SwapFee)
	params.Reserves = cloneU(params.Reserves)
	params.TotalSupply = cloneU(params.TotalSupply)
	params.ConvexityExp = cloneU(params.ConvexityExp)
	params.LastInvariant = cloneU(params.LastInvariant)
	return params
}

func cloneU(x *uint256.Int) *uint256.Int {
	if x == nil {
		return nil
	}
	return x.Clone()
}
