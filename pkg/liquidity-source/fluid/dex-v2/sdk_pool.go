package dexv2

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var SIX_DECIMALS = new(utils.Uint256).SetUint64(1_000_000)

type UniV3FluidV2Pool struct {
	v3Entities.Pool

	tickSpacing           uint32
	protocolFeeZeroForOne *utils.Uint128
	protocolFeeOneForZero *utils.Uint128
	constantLpFee         *utils.Uint128
	dynamicPacked         uint256.Int
	dynamicVersion        bool
	snapshotTimestamp     uint64
	protocolCutFee        uint64
}

type SwapResult struct {
	amountCalculated   *utils.Int256
	sqrtRatioX96       *utils.Uint160
	liquidity          *utils.Uint128
	remainingAmountIn  *utils.Int256
	currentTick        int
	crossInitTickLoops int
	grossOut           utils.Uint256
	dynamicPacked      uint256.Int
	protocolFeeOut     uint256.Int
	lpFeeOut           uint256.Int
	sqrtPriceReference uint256.Int
}

type StepComputations struct {
	sqrtPriceStartX96 utils.Uint160
	tickNext          int
	initialized       bool
	sqrtPriceNextX96  utils.Uint160
	amountIn          utils.Uint256
	amountOut         utils.Uint256
	feeAmount         utils.Uint256
}

func NewUniV3FluidV2Pool(
	tokenA, tokenB *entities.Token, fee constants.FeeAmount, sqrtRatioX96 *utils.Uint160,
	liquidity *utils.Uint128, tickCurrent int, ticks v3Entities.TickDataProvider,
	tickSpacing uint32, dexVariables2 *big.Int,
) (*UniV3FluidV2Pool, error) {
	v3Pool, err := v3Entities.NewPoolV2(
		tokenA,
		tokenB,
		fee,
		sqrtRatioX96,
		liquidity,
		tickCurrent,
		ticks,
	)

	if err != nil {
		return nil, err
	}

	pool := &UniV3FluidV2Pool{
		Pool:                  *v3Pool,
		tickSpacing:           tickSpacing,
		protocolFeeZeroForOne: new(utils.Uint128),
		protocolFeeOneForZero: new(utils.Uint128),
		constantLpFee:         new(utils.Uint128),
	}

	var tmp big.Int

	tmp.Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_PROTOCOL_FEE_0_TO_1).And(&tmp, X12)
	pool.protocolFeeZeroForOne.SetFromBig(&tmp)

	tmp.Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_PROTOCOL_FEE_1_TO_0).And(&tmp, X12)
	pool.protocolFeeOneForZero.SetFromBig(&tmp)

	tmp.Set(dexVariables2).Rsh(&tmp, BITS_DEX_V2_VARIABLES2_LP_FEE).And(&tmp, X16)
	pool.constantLpFee.SetFromBig(&tmp)
	tmp.Set(dexVariables2).Rsh(&tmp, 24).And(&tmp, big.NewInt(63))
	pool.protocolCutFee = tmp.Uint64()

	return pool, nil
}

type fluidAmountResult struct {
	v3Entities.GetAmountResultV2
	grossOut           utils.Uint256
	dynamicPacked      uint256.Int
	protocolFeeOut     uint256.Int
	lpFeeOut           uint256.Int
	sqrtPriceReference uint256.Int
}

func (p *UniV3FluidV2Pool) GetOutputAmountV2(inputAmount *utils.Int256, zeroForOne bool,
	sqrtPriceLimitX96 *utils.Uint160) (*fluidAmountResult, error) {
	swapResult, err := p.swap(zeroForOne, inputAmount, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	return &fluidAmountResult{
		sqrtPriceReference: swapResult.sqrtPriceReference,
		grossOut:           swapResult.grossOut,
		dynamicPacked:      swapResult.dynamicPacked,
		protocolFeeOut:     swapResult.protocolFeeOut,
		lpFeeOut:           swapResult.lpFeeOut,
		GetAmountResultV2: v3Entities.GetAmountResultV2{
			ReturnedAmount:     new(utils.Int256).Neg(swapResult.amountCalculated),
			RemainingAmountIn:  new(utils.Int256).Set(swapResult.remainingAmountIn),
			SqrtRatioX96:       swapResult.sqrtRatioX96,
			Liquidity:          swapResult.liquidity,
			CurrentTick:        swapResult.currentTick,
			CrossInitTickLoops: swapResult.crossInitTickLoops,
		}}, nil
}

func (p *UniV3FluidV2Pool) swap(zeroForOne bool, amountSpecified *utils.Int256,
	sqrtPriceLimitX96 *utils.Uint160) (*SwapResult, error) {
	if amountSpecified.Sign() <= 0 {
		return nil, ErrAmountOutOfLimits
	}
	var err error
	if sqrtPriceLimitX96 == nil {
		if zeroForOne {
			sqrtPriceLimitX96 = new(uint256.Int).AddUint64(utils.MinSqrtRatioU256, 1)
		} else {
			sqrtPriceLimitX96 = new(uint256.Int).SubUint64(utils.MaxSqrtRatioU256, 1)
		}
	}

	if zeroForOne {
		if sqrtPriceLimitX96.Cmp(utils.MinSqrtRatioU256) < 0 {
			return nil, v3Entities.ErrSqrtPriceLimitX96TooLow
		}
		if sqrtPriceLimitX96.Cmp(p.SqrtRatioX96) >= 0 {
			return nil, v3Entities.ErrSqrtPriceLimitX96TooHigh
		}
	} else {
		if sqrtPriceLimitX96.Cmp(utils.MaxSqrtRatioU256) > 0 {
			return nil, v3Entities.ErrSqrtPriceLimitX96TooHigh
		}
		if sqrtPriceLimitX96.Cmp(p.SqrtRatioX96) <= 0 {
			return nil, v3Entities.ErrSqrtPriceLimitX96TooLow
		}
	}

	var sqrtPriceReference uint256.Int
	sqrtPriceReference.Set(p.SqrtRatioX96)
	referenceChanged := false

	// keep track of swap state

	state := struct {
		amountSpecifiedRemaining *utils.Int256
		amountCalculated         *utils.Int256
		sqrtPriceX96             *utils.Uint160
		tick                     int
		liquidity                *utils.Uint128
	}{
		amountSpecifiedRemaining: new(utils.Int256).Set(amountSpecified),
		amountCalculated:         int256.NewInt(0),
		sqrtPriceX96:             new(utils.Uint160).Set(p.SqrtRatioX96),
		tick:                     p.TickCurrent,
		liquidity:                new(utils.Uint128).Set(p.Liquidity),
	}

	// crossInitTickLoops is the number of loops that cross an initialized tick.
	// We only count when tick passes an initialized tick, since gas only significant in this case.
	crossInitTickLoops := 0
	var grossOut, protocolFeeOut, lpFeeOut utils.Uint256
	var dynamic dynamicFeeState
	if p.dynamicVersion {
		dynamic, err = prepareDynamicFee(&p.dynamicPacked, state.sqrtPriceX96, zeroForOne, p.snapshotTimestamp)
		if err != nil {
			return nil, err
		}
	}

	// start swap while loop
	for !state.amountSpecifiedRemaining.IsZero() && state.sqrtPriceX96.Cmp(sqrtPriceLimitX96) != 0 {
		var step StepComputations
		lpFeeBefore := lpFeeOut
		step.sqrtPriceStartX96.Set(state.sqrtPriceX96)

		// because each iteration of the while loop rounds, we can't optimize this code (relative to the smart contract)
		// by simply traversing to the next available tick, we instead need to exactly replicate
		// tickBitmap.nextInitializedTickWithinOneWord
		step.tickNext, step.initialized, err = p.TickDataProvider.NextInitializedTickWithinOneWord(state.tick, zeroForOne, int(p.tickSpacing))
		if err != nil {
			return nil, err
		}

		if step.tickNext <= MIN_TICK || step.tickNext >= MAX_TICK {
			return nil, ErrNextTickOutOfBounds
		}

		err = utils.GetSqrtRatioAtTickV2(step.tickNext, &step.sqrtPriceNextX96)
		if err != nil {
			return nil, err
		}

		if (zeroForOne && step.sqrtPriceNextX96.Cmp(state.sqrtPriceX96) > 0) ||
			(!zeroForOne && step.sqrtPriceNextX96.Cmp(state.sqrtPriceX96) < 0) {
			state.sqrtPriceX96.Set(&step.sqrtPriceNextX96)
		}

		if p.dynamicVersion {
			var protocolFee, next, net, gross utils.Uint256
			protocolFee.Set(lo.Ternary(zeroForOne, p.protocolFeeZeroForOne, p.protocolFeeOneForZero))
			if dynamic.factor.IsZero() {
				var unused utils.Uint256
				if err = utils.ComputeSwapStep(state.sqrtPriceX96, &step.sqrtPriceNextX96, state.liquidity, state.amountSpecifiedRemaining, 0, &next, &step.amountIn, &gross, &unused); err != nil {
					return nil, err
				}
				if gross.BitLen() > 86 {
					return nil, ErrOverflow
				}
				var protocolCharge, lpCharge utils.Uint256
				netSwapInStep(&net, &protocolCharge, &lpCharge, &gross, &protocolFee, &dynamic.minFee)
				if _, overflow := protocolFeeOut.AddOverflow(&protocolFeeOut, &protocolCharge); overflow {
					return nil, ErrOverflow
				}
				if _, overflow := lpFeeOut.AddOverflow(&lpFeeOut, &lpCharge); overflow {
					return nil, ErrOverflow
				}
			} else {
				if err = dynamicSwapStep(&dynamic, zeroForOne, state.sqrtPriceX96, &step.sqrtPriceNextX96, state.liquidity, state.amountSpecifiedRemaining, &protocolFee, &next, &step.amountIn, &net, &gross, &protocolFeeOut, &lpFeeOut); err != nil {
					return nil, err
				}
			}
			if _, overflow := grossOut.AddOverflow(&grossOut, &gross); overflow {
				return nil, ErrOverflow
			}
			state.sqrtPriceX96.Set(&next)
			var amountInSigned, amountOutSigned utils.Int256
			if err = utils.ToInt256(&step.amountIn, &amountInSigned); err != nil {
				return nil, err
			}
			if err = utils.ToInt256(&net, &amountOutSigned); err != nil {
				return nil, err
			}
			state.amountSpecifiedRemaining.Sub(state.amountSpecifiedRemaining, &amountInSigned)
			state.amountCalculated.Sub(state.amountCalculated, &amountOutSigned)
		} else {
			var nxtSqrtPriceX96 utils.Uint160
			err = utils.ComputeSwapStep(state.sqrtPriceX96, &step.sqrtPriceNextX96, state.liquidity, state.amountSpecifiedRemaining,
				0, // _computeSwapStepForSwapInWithoutFee
				&nxtSqrtPriceX96, &step.amountIn, &step.amountOut, &step.feeAmount)
			if err != nil {
				return nil, err
			}

			if step.amountOut.BitLen() > 86 {
				return nil, ErrOverflow
			}
			if _, overflow := grossOut.AddOverflow(&grossOut, &step.amountOut); overflow {
				return nil, ErrOverflow
			}
			var stepProtocolFee, stepLpFee, protocolFee utils.Uint256
			protocolFee.Set(lo.Ternary(zeroForOne,
				p.protocolFeeZeroForOne,
				p.protocolFeeOneForZero,
			))
			// Deployed swap module charges protocol first, then LP on the remainder.
			// Each non-zero fee rounds as ((amount * fee + 1) / 1e6) + 1.
			// The contract bounds step output to 86 bits and the fee to 16 bits.
			var amountOutPlusFee utils.Uint256
			netSwapInStep(&amountOutPlusFee, &stepProtocolFee, &stepLpFee, &step.amountOut, &protocolFee, p.constantLpFee)

			state.sqrtPriceX96.Set(&nxtSqrtPriceX96)
			var amountInSigned utils.Int256
			err = utils.ToInt256(&step.amountIn, &amountInSigned)
			if err != nil {
				return nil, err
			}

			var amountOutSigned utils.Int256
			err = utils.ToInt256(&amountOutPlusFee, &amountOutSigned)
			if err != nil {
				return nil, err
			}

			state.amountSpecifiedRemaining.Sub(state.amountSpecifiedRemaining, &amountInSigned)
			state.amountCalculated.Sub(state.amountCalculated, &amountOutSigned)

			if _, overflow := protocolFeeOut.AddOverflow(&protocolFeeOut, &stepProtocolFee); overflow {
				return nil, ErrOverflow
			}
			if _, overflow := lpFeeOut.AddOverflow(&lpFeeOut, &stepLpFee); overflow {
				return nil, ErrOverflow
			}
		}
		// The protocol cut is applied once per tick step, after combining the
		// dynamic fee zones. It affects the separate fee-token conversions.
		if p.protocolCutFee != 0 {
			var lpStep, cut, rate, hundred uint256.Int
			lpStep.Sub(&lpFeeOut, &lpFeeBefore)
			if !lpStep.IsZero() {
				rate.SetUint64(p.protocolCutFee)
				hundred.SetUint64(100)
				// Step fees fit the 86-bit gross output bound; cut is at most 63.
				cut.Mul(&lpStep, &rate).AddUint64(&cut, 1).Div(&cut, &hundred).AddUint64(&cut, 1)
				if cut.Gt(&lpStep) {
					cut = lpStep
				}
				lpFeeOut.Sub(&lpFeeOut, &cut)
				if _, overflow := protocolFeeOut.AddOverflow(&protocolFeeOut, &cut); overflow {
					return nil, ErrOverflow
				}
			}
		}
		if state.sqrtPriceX96.Cmp(&step.sqrtPriceNextX96) == 0 {
			// if the tick is initialized, run the tick transition
			if step.initialized {
				tick, err := p.TickDataProvider.GetTick(step.tickNext)
				if err != nil {
					return nil, err
				}

				liquidityNet := tick.LiquidityNet
				// if we're moving leftward, we interpret liquidityNet as the opposite sign
				// safe because liquidityNet cannot be type(int128).min
				if zeroForOne {
					liquidityNet = new(utils.Int128).Neg(liquidityNet)
				}
				err = utils.AddDeltaInPlace(state.liquidity, liquidityNet)
				if err != nil {
					return nil, err
				}

				// Only swaps starting without liquidity re-base the price-move reference.
				// The dynamic fee anchor deliberately keeps its original price.
				if p.Liquidity.IsZero() && !referenceChanged && !state.liquidity.IsZero() {
					sqrtPriceReference.Set(state.sqrtPriceX96)
					referenceChanged = true
				}

				crossInitTickLoops++
			}
			if zeroForOne {
				state.tick = step.tickNext - 1
			} else {
				state.tick = step.tickNext
			}

		} else if state.sqrtPriceX96.Cmp(&step.sqrtPriceStartX96) != 0 {
			// recompute unless we're on a lower tick boundary (i.e. already transitioned ticks), and haven't moved
			state.tick, err = utils.GetTickAtSqrtRatioV2(state.sqrtPriceX96)
			if err != nil {
				return nil, err
			}
		}
	}

	var nextDynamicPacked uint256.Int
	if p.dynamicVersion {
		nextDynamicPacked, err = finishDynamicFee(&dynamic, state.sqrtPriceX96)
		if err != nil {
			return nil, err
		}
		// The on-chain swap also stores the final active liquidity in bits 38..139.
		if state.liquidity.BitLen() > 102 {
			return nil, ErrOverflow
		}
		var mask, active uint256.Int
		mask.SetOne().Lsh(&mask, 102).SubUint64(&mask, 1).Lsh(&mask, 38).Not(&mask)
		active.Lsh(state.liquidity, 38)
		nextDynamicPacked.And(&nextDynamicPacked, &mask).Or(&nextDynamicPacked, &active)
	}
	return &SwapResult{
		amountCalculated:   state.amountCalculated,
		sqrtRatioX96:       state.sqrtPriceX96,
		liquidity:          state.liquidity,
		currentTick:        state.tick,
		remainingAmountIn:  state.amountSpecifiedRemaining,
		crossInitTickLoops: crossInitTickLoops,
		grossOut:           grossOut,
		dynamicPacked:      nextDynamicPacked,
		protocolFeeOut:     protocolFeeOut,
		lpFeeOut:           lpFeeOut,
		sqrtPriceReference: sqrtPriceReference,
	}, nil
}

// netSwapInStep applies the deployed protocol and LP fee order and rounding.
// The caller bounds gross to 86 bits, protocol fee to 12 bits and LP fee to 16 bits.
func netSwapInStep(net, protocolCharge, lpCharge, gross, protocolFee, lpFee *utils.Uint256) {
	net.Set(gross)
	if !net.IsZero() && !protocolFee.IsZero() {
		protocolCharge.Mul(net, protocolFee).AddUint64(protocolCharge, 1).Div(protocolCharge, SIX_DECIMALS).AddUint64(protocolCharge, 1)
		if net.Cmp(protocolCharge) > 0 {
			net.Sub(net, protocolCharge)
		} else {
			net.Clear()
		}
	}
	if !net.IsZero() && !lpFee.IsZero() {
		lpCharge.Mul(net, lpFee).AddUint64(lpCharge, 1).Div(lpCharge, SIX_DECIMALS).AddUint64(lpCharge, 1)
		if net.Cmp(lpCharge) > 0 {
			net.Sub(net, lpCharge)
		} else {
			net.Clear()
		}
	}
}
