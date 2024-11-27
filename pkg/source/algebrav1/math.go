package algebrav1

import (
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

type SwapCalculationCache struct {
	communityFee     uint256.Int    // The community fee of the selling token, uint256 to minimize casts
	amountRequired   v3Utils.Int256 // The required value of the exact input\output amount
	amountCalculated v3Utils.Int256 // The additive amount of total output\input calculated through the swap
	exactInput       bool           // Whether the exact input or output is specified
	fee              uint16         // The current dynamic fee
}

type PriceMovementCache struct {
	stepSqrtPrice v3Utils.Uint160 // The Q64.96 sqrt of the price at the start of the step
	nextTick      int             // The tick till the current step goes
	initialized   bool            // True if the _nextTick is initialized
	nextTickPrice v3Utils.Uint160 // The Q64.96 sqrt of the price calculated from the _nextTick
	input         uint256.Int     // The additive amount of tokens that have been provided
	output        uint256.Int     // The additive amount of token that have been withdrawn
	feeAmount     uint256.Int     // The total amount of fee earned within a current step
}

type SwapResult struct {
	*StateUpdate
	amountCalculated   *v3Utils.Int256
	remainingAmountIn  *v3Utils.Int256
	crossInitTickLoops int64
}

// https://github.com/cryptoalgebra/AlgebraV1/blob/dfebf532a27803dafcbf2ba49724740bd6220505/src/core/contracts/AlgebraPool.sol#L703
func (p *PoolSimulator) _calculateSwapAndLock(
	zeroToOne bool,
	amountRequired *v3Utils.Int256,
	limitSqrtPrice *v3Utils.Uint160,
) (*SwapResult, error) {
	var cache SwapCalculationCache
	var err error

	nextState := &StateUpdate{}

	var currentPrice v3Utils.Uint160
	currentPrice.Set(p.globalState.Price)
	currentTick := p.globalState.Tick
	_communityFeeToken0 := p.globalState.CommunityFeeToken0
	_communityFeeToken1 := p.globalState.CommunityFeeToken1

	cmp := amountRequired.Sign()
	if cmp == 0 {
		return nil, ErrZeroAmountIn
	}

	cache.amountRequired, cache.exactInput = *amountRequired, cmp > 0

	currentLiquidity := p.liquidity

	if zeroToOne {
		if limitSqrtPrice.Cmp(&currentPrice) >= 0 || limitSqrtPrice.Cmp(v3Utils.MinSqrtRatioU256) <= 0 {
			return nil, ErrSPL
		}
		cache.communityFee.SetUint64(uint64(_communityFeeToken0))
	} else {
		if limitSqrtPrice.Cmp(&currentPrice) <= 0 || limitSqrtPrice.Cmp(v3Utils.MaxSqrtRatioU256) >= 0 {
			return nil, ErrSPL
		}
		cache.communityFee.SetUint64(uint64(_communityFeeToken1))
	}

	// don't need to care about activeIncentive

	// use pre-calculated fee instead of calculating from timepoints
	// see tracker code for more details
	if zeroToOne {
		cache.fee = p.globalState.FeeZto
	} else {
		cache.fee = p.globalState.FeeOtz
	}

	var crossInitTickLoops int64
	var step PriceMovementCache
	// swap until there is remaining input or output tokens, or we reach the price limit.
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		step.stepSqrtPrice = currentPrice

		if step.nextTick, step.initialized, err = p.ticks.NextInitializedTickWithinOneWord(currentTick, zeroToOne,
			p.tickSpacing); err != nil {
			return nil, err
		}

		if err := v3Utils.GetSqrtRatioAtTickV2(step.nextTick, &step.nextTickPrice); err != nil {
			return nil, err
		}

		// calculate the amounts needed to move the price to the next target if it is possible or as much as possible
		targetPrice := &step.nextTickPrice
		if zeroToOne == (step.nextTickPrice.Cmp(limitSqrtPrice) < 0) {
			targetPrice = limitSqrtPrice
		}

		var nxtSqrtPriceX96 v3Utils.Uint160
		if err = v3Utils.ComputeSwapStep(
			&currentPrice, targetPrice, currentLiquidity, &cache.amountRequired, constants.FeeAmount(cache.fee),
			&nxtSqrtPriceX96, &step.input, &step.output, &step.feeAmount,
		); err != nil {
			return nil, err
		}
		currentPrice.Set(&nxtSqrtPriceX96)

		var amountInPlusFee v3Utils.Uint256
		amountInPlusFee.Add(&step.input, &step.feeAmount)

		var amountInPlusFeeSigned v3Utils.Int256
		if err := v3Utils.ToInt256(&amountInPlusFee, &amountInPlusFeeSigned); err != nil {
			return nil, err
		}

		var amountOutSigned v3Utils.Int256
		if err := v3Utils.ToInt256(&step.output, &amountOutSigned); err != nil {
			return nil, err
		}

		if cache.exactInput {
			cache.amountRequired.Sub(&cache.amountRequired,
				&amountInPlusFeeSigned) // decrease remaining input amount
			cache.amountCalculated.Sub(&cache.amountCalculated,
				&amountOutSigned) // decrease calculated output amount
		} else {
			cache.amountRequired.Add(&cache.amountRequired,
				&amountOutSigned) // increase remaining output amount (since its negative)
			cache.amountCalculated.Add(&cache.amountCalculated,
				&amountInPlusFeeSigned,
			) // increase calculated input amount
		}

		if cache.communityFee.Sign() > 0 {
			delta := amountInPlusFee.Div(
				amountInPlusFee.Mul(&step.feeAmount, &cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)
			step.feeAmount.Sub(&step.feeAmount, delta)
		}

		if currentPrice == step.nextTickPrice {
			// if the reached tick is initialized then we need to cross it
			if step.initialized {
				// once at a swap we have to get the last timepoint of the observation
				// don't need to do this here

				// every tick cross is needed to be duplicated in a virtual pool
				// don't need to do this here

				nextTickData, err := p.ticks.GetTick(step.nextTick)
				if err != nil {
					return nil, err
				}
				liquidityDelta := nextTickData.LiquidityNet
				if zeroToOne {
					liquidityDelta = new(v3Utils.Int128).Neg(nextTickData.LiquidityNet)
				}

				_ = v3Utils.AddDeltaInPlace(currentLiquidity, liquidityDelta)
				crossInitTickLoops++
			}
			if zeroToOne {
				currentTick = step.nextTick - 1
			} else {
				currentTick = step.nextTick
			}
		} else if currentPrice != step.stepSqrtPrice {
			// if the price has changed but hasn't reached the target
			if currentTick, err = v3Utils.GetTickAtSqrtRatioV2(&currentPrice); err != nil {
				return nil, err
			}
			break // since the price hasn't reached the target, amountRequired should be 0
		}

		// check stop condition
		if cache.amountRequired.IsZero() || currentPrice.Cmp(limitSqrtPrice) == 0 {
			break
		}
	}

	nextState.GlobalState = GlobalStateUint256{
		Price:              &currentPrice,
		Tick:               currentTick,
		FeeZto:             p.globalState.FeeZto,
		FeeOtz:             p.globalState.FeeOtz,
		TimepointIndex:     p.globalState.TimepointIndex,
		CommunityFeeToken0: p.globalState.CommunityFeeToken0,
		CommunityFeeToken1: p.globalState.CommunityFeeToken0,
	}

	nextState.Liquidity = currentLiquidity

	return &SwapResult{
		StateUpdate:        nextState,
		amountCalculated:   &cache.amountCalculated,
		remainingAmountIn:  &cache.amountRequired,
		crossInitTickLoops: crossInitTickLoops,
	}, nil
}
