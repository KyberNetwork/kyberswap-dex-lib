package lunya

import (
	"github.com/holiman/uint256"
)

type stableSwapResult struct {
	// amountIn is what the caller pays, fee included; amountOut what it receives.
	amountIn      uint256.Int
	amountOut     uint256.Int
	curveReserve0 uint256.Int
	curveReserve1 uint256.Int
	sqrtPriceX96  uint256.Int
}

// swap is LunyaPoolSTABLE.swap's pricing: amounts are lifted to 18 decimals, moved along the curve,
// and brought back rounding the charge up and the payout down. amount is the input for exact input and
// the requested output otherwise.
func (p *PoolSimulator) stableSwap(zeroForOne, exactInput bool, amount, sqrtPriceLimitX96 *uint256.Int) (*stableSwapResult,
	error) {
	switch {
	case p.halted:
		return nil, ErrTradingHalted
	case p.liquidity.IsZero():
		return nil, ErrNoLiquidity
	case p.fee >= feeDenominator:
		return nil, ErrInvalidFee
	}

	if zeroForOne {
		if !sqrtPriceLimitX96.Lt(&p.sqrtPriceX96) || !sqrtPriceLimitX96.Gt(minSqrtRatio) {
			return nil, ErrInvalidSqrtPriceLimit
		}
	} else if !sqrtPriceLimitX96.Gt(&p.sqrtPriceX96) || !sqrtPriceLimitX96.Lt(maxSqrtRatio) {
		return nil, ErrInvalidSqrtPriceLimit
	}

	feeOnOutput := p.feeOnOutput(zeroForOne)
	var fee, feeComplement uint256.Int
	fee.SetUint64(uint64(p.fee))
	feeComplement.SetUint64(feeDenominator - uint64(p.fee))

	var curve0, curve1 uint256.Int
	if _, overflow := curve0.MulOverflow(&p.curveReserve0, &p.rate0); overflow {
		return nil, ErrArithmetic
	}
	if _, overflow := curve1.MulOverflow(&p.curveReserve1, &p.rate1); overflow {
		return nil, ErrArithmetic
	}
	rateInput, rateOutput := &p.rate0, &p.rate1
	if !zeroForOne {
		rateInput, rateOutput = rateOutput, rateInput
	}

	magnitude := *amount
	scale := rateInput
	if !exactInput {
		scale = rateOutput
		// an exact output on the side the fee is taken from is grossed up to cover it
		if feeOnOutput {
			if err := mulDivRoundingUp(&magnitude, &magnitude, feeDenominatorU, &feeComplement); err != nil {
				return nil, err
			}
		}
	}
	var bound, amountAvailable uint256.Int
	if magnitude.Gt(bound.Div(maxInt256, scale)) {
		return nil, ErrInvalidAmount
	}
	amountAvailable.Mul(&magnitude, scale)
	if !exactInput {
		amountAvailable.Neg(&amountAvailable)
	}

	stepFee := uint64(p.fee)
	if feeOnOutput {
		stepFee = 0
	}
	var ann uint256.Int
	ann.SetUint64(p.amplificationAt(nowUnix()) * 4)

	targetPrice, err := p.scalePrice(sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	resultPrice, inN, outN, feeN, err := movePriceTowardsTarget(zeroForOne, &targetPrice, &curve0, &curve1,
		&amountAvailable, stepFee, &ann)
	if err != nil {
		return nil, err
	}

	var input, output, feeAmount uint256.Int
	if feeOnOutput {
		// the curve ran fee-free; the fee comes off the gross output, rounded up
		divUp(&input, &inN, rateInput)
		var gross uint256.Int
		gross.Div(&outN, rateOutput)
		if err := mulDivRoundingUp(&feeAmount, &gross, &fee, feeDenominatorU); err != nil {
			return nil, err
		}
		output.Sub(&gross, &feeAmount)
	} else {
		// the gross is rounded up once, and the fee keeps its own rounding
		divUp(&feeAmount, &feeN, rateInput)
		var gross uint256.Int
		if _, overflow := gross.AddOverflow(&inN, &feeN); overflow {
			return nil, ErrArithmetic
		}
		divUp(&input, &gross, rateInput)
		if input.Lt(&feeAmount) {
			return nil, ErrArithmetic
		}
		input.Sub(&input, &feeAmount)
		output.Div(&outN, rateOutput)
	}

	s := &stableSwapResult{curveReserve0: p.curveReserve0, curveReserve1: p.curveReserve1}
	if s.sqrtPriceX96, err = p.unscalePrice(&resultPrice); err != nil {
		return nil, err
	}

	// the net input joins the curve and the gross output leaves it; the fee stays in the pool as balance
	fromCurve := output
	s.amountIn = input
	if feeOnOutput {
		fromCurve.Add(&fromCurve, &feeAmount)
	} else {
		s.amountIn.Add(&s.amountIn, &feeAmount)
	}
	s.amountOut = output

	reserveIn, reserveOut := &s.curveReserve0, &s.curveReserve1
	if !zeroForOne {
		reserveIn, reserveOut = reserveOut, reserveIn
	}
	if reserveIn.Add(reserveIn, &input).Gt(maxUint128) || reserveOut.Lt(&fromCurve) {
		return nil, ErrArithmetic
	}
	reserveOut.Sub(reserveOut, &fromCurve)

	return s, nil
}

// amplificationAt is LunyaPoolSTABLE._scaledAmplification at the given time: A in hundredths, moved
// linearly along a ramp in flight.
func (p *PoolSimulator) amplificationAt(now uint64) uint64 {
	r := p.ramp
	if r == nil {
		return uint64(p.amplificationX100)
	}
	if now >= uint64(r.EndTime) {
		return uint64(r.TargetAmplification)
	}

	elapsed := max(now, uint64(r.StartTime)) - uint64(r.StartTime)
	span := uint64(r.EndTime) - uint64(r.StartTime)
	if r.TargetAmplification > r.StartAmplification {
		rise := uint64(r.TargetAmplification - r.StartAmplification)
		return uint64(r.StartAmplification) + uint64(uint32(rise*elapsed/span))
	}
	fall := uint64(r.StartAmplification - r.TargetAmplification)
	return uint64(r.StartAmplification) - uint64(uint32(fall*elapsed/span))
}

// scalePrice lifts a raw sqrt price into the normalised space the curve works in, saturating.
func (p *PoolSimulator) scalePrice(rawSqrtPrice *uint256.Int) (uint256.Int, error) {
	var v uint256.Int
	if err := mulDiv(&v, rawSqrtPrice, &p.priceScaleSqrtQ96, q96); err != nil {
		return v, err
	}
	if v.IsZero() {
		return *one, nil
	}
	if v.Gt(maxUint160) {
		return *maxUint160, nil
	}
	return v, nil
}

// unscalePrice is scalePrice's inverse, clamped into the range a Q64.96 price may take.
func (p *PoolSimulator) unscalePrice(scaledSqrtPrice *uint256.Int) (uint256.Int, error) {
	var v uint256.Int
	if err := mulDiv(&v, scaledSqrtPrice, q96, &p.priceScaleSqrtQ96); err != nil {
		return v, err
	}
	if v.Lt(minSqrtRatio) {
		return *minSqrtRatio, nil
	}
	if !v.Lt(maxSqrtRatio) {
		return *v.SubUint64(maxSqrtRatio, 1), nil
	}
	return v, nil
}

// divUp is ceiling division, zero for zero.
func divUp(z, a, b *uint256.Int) {
	if a.IsZero() {
		z.Clear()
		return
	}
	z.SubUint64(a, 1).Div(z, b).AddUint64(z, 1)
}
