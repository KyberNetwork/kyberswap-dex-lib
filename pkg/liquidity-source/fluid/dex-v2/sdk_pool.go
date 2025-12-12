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
}

type SwapResult struct {
	amountCalculated   *utils.Int256
	sqrtRatioX96       *utils.Uint160
	liquidity          *utils.Uint128
	remainingAmountIn  *utils.Int256
	currentTick        int
	crossInitTickLoops int
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

	return pool, nil
}

func (p *UniV3FluidV2Pool) GetOutputAmountV2(inputAmount *utils.Int256, zeroForOne bool,
	sqrtPriceLimitX96 *utils.Uint160) (*v3Entities.GetAmountResultV2, error) {
	swapResult, err := p.swap(zeroForOne, inputAmount, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	return &v3Entities.GetAmountResultV2{
		ReturnedAmount:     new(utils.Int256).Neg(swapResult.amountCalculated),
		RemainingAmountIn:  new(utils.Int256).Set(swapResult.remainingAmountIn),
		SqrtRatioX96:       swapResult.sqrtRatioX96,
		Liquidity:          swapResult.liquidity,
		CurrentTick:        swapResult.currentTick,
		CrossInitTickLoops: swapResult.crossInitTickLoops,
	}, nil
}

func (p *UniV3FluidV2Pool) swap(zeroForOne bool, amountSpecified *utils.Int256,
	sqrtPriceLimitX96 *utils.Uint160) (*SwapResult, error) {
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

	exactInput := amountSpecified.Sign() >= 0

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

	// start swap while loop
	for !state.amountSpecifiedRemaining.IsZero() && state.sqrtPriceX96.Cmp(sqrtPriceLimitX96) != 0 {
		var step StepComputations
		step.sqrtPriceStartX96.Set(state.sqrtPriceX96)

		// because each iteration of the while loop rounds, we can't optimize this code (relative to the smart contract)
		// by simply traversing to the next available tick, we instead need to exactly replicate
		// tickBitmap.nextInitializedTickWithinOneWord
		step.tickNext, step.initialized, err = p.TickDataProvider.NextInitializedTickWithinOneWord(state.tick, zeroForOne, int(p.tickSpacing))
		if err != nil {
			return nil, err
		}

		if step.tickNext < utils.MinTick {
			step.tickNext = utils.MinTick
		} else if step.tickNext > utils.MaxTick {
			step.tickNext = utils.MaxTick
		}

		err = utils.GetSqrtRatioAtTickV2(step.tickNext, &step.sqrtPriceNextX96)
		if err != nil {
			return nil, err
		}

		if (zeroForOne && step.sqrtPriceNextX96.Cmp(state.sqrtPriceX96) > 0) ||
			(!zeroForOne && step.sqrtPriceNextX96.Cmp(state.sqrtPriceX96) < 0) {
			state.sqrtPriceX96.Set(&step.sqrtPriceNextX96)
		}

		var nxtSqrtPriceX96 utils.Uint160
		// TODO: support _computeSwapStepForSwapInWithDynamicFee
		err = utils.ComputeSwapStep(state.sqrtPriceX96, &step.sqrtPriceNextX96, state.liquidity, state.amountSpecifiedRemaining,
			0, // _computeSwapStepForSwapInWithoutFee
			&nxtSqrtPriceX96, &step.amountIn, &step.amountOut, &step.feeAmount)
		if err != nil {
			return nil, err
		}

		var stepProtocolFee, stepLpFee, protocolFee utils.Uint256
		protocolFee.Set(lo.Ternary(zeroForOne,
			p.protocolFeeZeroForOne,
			p.protocolFeeOneForZero,
		))
		if exactInput {
			// Fluid v2 custom logic: calculate protocol fee and lp fee on amountOut
			stepProtocolFee.Mul(&step.amountOut, &protocolFee).
				Div(&stepProtocolFee, SIX_DECIMALS)

			stepLpFee.Mul(&step.amountOut, p.constantLpFee).
				Div(&stepLpFee, SIX_DECIMALS)

			state.sqrtPriceX96.Set(&nxtSqrtPriceX96)

			// Fluid v2 custom logic: apply fee on amountOut instead of amountIn as in Uniswap v3
			var amountInSigned utils.Int256
			err = utils.ToInt256(&step.amountIn, &amountInSigned)
			if err != nil {
				return nil, err
			}

			var amountOutPlusFee utils.Uint256
			amountOutPlusFee.Sub(&step.amountOut, &stepLpFee)

			var amountOutSigned utils.Int256
			err = utils.ToInt256(&amountOutPlusFee, &amountOutSigned)
			if err != nil {
				return nil, err
			}

			state.amountSpecifiedRemaining.Sub(state.amountSpecifiedRemaining, &amountInSigned)
			state.amountCalculated.Sub(state.amountCalculated, &amountOutSigned)
		} else {
			// TODO: Test this branch
			var amountInWithFee utils.Uint256
			amountInWithFee.Mul(&step.amountIn, SIX_DECIMALS).
				Div(&amountInWithFee, new(utils.Uint256).Sub(SIX_DECIMALS, p.constantLpFee))
			stepLpFee.Sub(&amountInWithFee, &step.amountIn)

			amountInWithFee.Mul(&amountInWithFee, SIX_DECIMALS).
				Div(&amountInWithFee, new(utils.Uint256).Sub(SIX_DECIMALS, &protocolFee))
			stepProtocolFee.Sub(&amountInWithFee, &step.amountIn).
				Sub(&stepProtocolFee, &stepLpFee)

			var amountInSigned utils.Int256
			err = utils.ToInt256(&amountInWithFee, &amountInSigned)
			if err != nil {
				return nil, err
			}

			var amountOutSigned utils.Int256
			err = utils.ToInt256(&step.amountOut, &amountOutSigned)
			if err != nil {
				return nil, err
			}

			state.amountSpecifiedRemaining.Add(state.amountSpecifiedRemaining, &amountOutSigned)
			state.amountCalculated.Add(state.amountCalculated, &amountInSigned)
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
				utils.AddDeltaInPlace(state.liquidity, liquidityNet)

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

	return &SwapResult{
		amountCalculated:   state.amountCalculated,
		sqrtRatioX96:       state.sqrtPriceX96,
		liquidity:          state.liquidity,
		currentTick:        state.tick,
		remainingAmountIn:  state.amountSpecifiedRemaining,
		crossInitTickLoops: crossInitTickLoops,
	}, nil
}
