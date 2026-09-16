package lunya

import (
	"sort"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

type clSwapResult struct {
	amountSpecifiedRemaining uint256.Int // int256, two's complement
	amountCalculated         uint256.Int // int256, two's complement
	sqrtPriceX96             uint256.Int
	liquidity                uint256.Int
	tick                     int
	crossedTicks             int
}

// swap is LunyaPoolCL.swap's pricing loop. amountSpecified is an int256 in two's complement: positive
// for exact input, negative for exact output.
//
// Two things separate it from Uniswap V3's loop, and both change amounts:
//   - the next tick is the next initialized one (TickTree), never a word boundary, so steps are not
//     split where V3's bitmap walk would split them;
//   - when the fee is charged in the output token, each step is computed fee-free and the fee is
//     taken off the step's output, rounded up; an exact-output request is grossed up to cover it.
func (p *PoolSimulator) clSwap(zeroForOne bool, amountSpecified, sqrtPriceLimitX96 *uint256.Int) (*clSwapResult, error) {
	if p.halted {
		return nil, ErrTradingHalted
	} else if p.fee >= feeDenominator {
		return nil, ErrInvalidFee
	}

	if zeroForOne {
		if !sqrtPriceLimitX96.Lt(&p.sqrtPriceX96) || !sqrtPriceLimitX96.Gt(minSqrtRatio) {
			return nil, ErrInvalidSqrtPriceLimit
		}
	} else if !sqrtPriceLimitX96.Gt(&p.sqrtPriceX96) || !sqrtPriceLimitX96.Lt(maxSqrtRatio) {
		return nil, ErrInvalidSqrtPriceLimit
	}

	exactInput := amountSpecified.Sign() > 0
	feeOnOutput := p.feeOnOutput(zeroForOne)

	stepFee := uniswapv3.FeeAmount(p.fee)
	if feeOnOutput {
		stepFee = 0
	}
	var fee, feeComplement uint256.Int
	fee.SetUint64(uint64(p.fee))
	feeComplement.SetUint64(feeDenominator - uint64(p.fee))

	s := &clSwapResult{
		amountSpecifiedRemaining: *amountSpecified,
		sqrtPriceX96:             p.sqrtPriceX96,
		liquidity:                p.liquidity,
		tick:                     p.tick,
	}

	for !s.amountSpecifiedRemaining.IsZero() && !s.sqrtPriceX96.Eq(sqrtPriceLimitX96) {
		sqrtPriceStartX96 := s.sqrtPriceX96

		tickNext, pos, initialized := p.nextTick(s.tick, zeroForOne)
		var sqrtPriceNextX96 uint256.Int
		switch {
		case initialized:
			sqrtPriceNextX96 = p.tickSqrtPrices[pos]
		case tickNext < uniswapv3.MinTick:
			tickNext, sqrtPriceNextX96 = uniswapv3.MinTick, *minSqrtRatio
		default:
			tickNext, sqrtPriceNextX96 = uniswapv3.MaxTick, *maxSqrtRatio
		}

		target := &sqrtPriceNextX96
		if zeroForOne && sqrtPriceNextX96.Lt(sqrtPriceLimitX96) || !zeroForOne && sqrtPriceNextX96.Gt(sqrtPriceLimitX96) {
			target = sqrtPriceLimitX96
		}

		remainingForStep := s.amountSpecifiedRemaining
		if feeOnOutput && !exactInput {
			var owed uint256.Int
			owed.Neg(&s.amountSpecifiedRemaining)
			if err := uniswapv3.MulDivRoundingUpV2(&owed, feeDenominatorU, &feeComplement,
				&remainingForStep); err != nil {
				return nil, err
			} else if remainingForStep.Sign() < 0 {
				return nil, ErrInt256Overflow
			}
			remainingForStep.Neg(&remainingForStep)
		}

		var sqrtPriceX96, amountIn, amountOut, feeAmount uint256.Int
		if err := uniswapv3.ComputeSwapStep(&s.sqrtPriceX96, target, &s.liquidity, &remainingForStep, stepFee,
			&sqrtPriceX96, &amountIn, &amountOut, &feeAmount); err != nil {
			return nil, err
		}
		s.sqrtPriceX96 = sqrtPriceX96

		var paidIn uint256.Int
		paidIn.Add(&amountIn, &feeAmount)

		if feeOnOutput {
			if err := uniswapv3.MulDivRoundingUpV2(&amountOut, &fee, feeDenominatorU, &feeAmount); err != nil {
				return nil, err
			}
			amountOut.Sub(&amountOut, &feeAmount)
		}

		if paidIn.Sign() < 0 || amountOut.Sign() < 0 {
			return nil, ErrInt256Overflow
		}
		if exactInput {
			s.amountSpecifiedRemaining.Sub(&s.amountSpecifiedRemaining, &paidIn)
			s.amountCalculated.Sub(&s.amountCalculated, &amountOut)
		} else {
			s.amountSpecifiedRemaining.Add(&s.amountSpecifiedRemaining, &amountOut)
			s.amountCalculated.Add(&s.amountCalculated, &paidIn)
		}

		if s.sqrtPriceX96.Eq(&sqrtPriceNextX96) {
			if initialized {
				liquidityNet := uint256.Int(*p.ticks[pos].LiquidityNet)
				if zeroForOne {
					liquidityNet.Neg(&liquidityNet)
				}
				s.liquidity.Add(&s.liquidity, &liquidityNet)
				if s.liquidity.Sign() < 0 || s.liquidity.BitLen() > 128 {
					return nil, ErrLiquidityOverflow
				}
				s.crossedTicks++
			}
			if zeroForOne {
				s.tick = tickNext - 1
			} else {
				s.tick = tickNext
			}
		} else if !s.sqrtPriceX96.Eq(&sqrtPriceStartX96) {
			tick, err := uniswapv3.GetTickAtSqrtRatio(&s.sqrtPriceX96)
			if err != nil {
				return nil, err
			}
			s.tick = tick
		}
	}

	return s, nil
}

// feeOnOutput is LunyaPoolBase._feeOnOutput.
func (p *PoolSimulator) feeOnOutput(zeroForOne bool) bool {
	switch p.feeToken {
	case feeTokenToken0:
		return !zeroForOne
	case feeTokenToken1:
		return zeroForOne
	default:
		return false
	}
}

// nextTick is TickTree.prevOrEq (zeroForOne: the largest initialized tick at or below tick) and
// TickTree.next (the smallest initialized tick strictly above it). When there is none it returns
// MIN_TICK - 1 or MAX_TICK + 1, which the swap loop clamps and treats as uninitialized.
func (p *PoolSimulator) nextTick(tick int, zeroForOne bool) (tickNext, pos int, initialized bool) {
	above := sort.Search(len(p.ticks), func(i int) bool { return p.ticks[i].Index > tick })
	if zeroForOne {
		if above == 0 {
			return uniswapv3.MinTick - 1, -1, false
		}
		return p.ticks[above-1].Index, above - 1, true
	}
	if above == len(p.ticks) {
		return uniswapv3.MaxTick + 1, -1, false
	}
	return p.ticks[above].Index, above, true
}
