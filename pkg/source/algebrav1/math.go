package algebrav1

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/daoleno/uniswapv3-sdk/utils"
)

type SwapCalculationCache struct {
	communityFee *big.Int // The community fee of the selling token, uint256 to minimize casts
	// volumePerLiquidityInBlock     *big.Int
	// tickCumulative                int64    // The global tickCumulative at the moment
	// secondsPerLiquidityCumulative *big.Int // The global secondPerLiquidity at the moment
	// computedLatestTimepoint       bool     //  if we have already fetched _tickCumulative_ and _secondPerLiquidity_ from the DataOperator
	amountRequiredInitial *big.Int // The initial value of the exact input\output amount
	amountCalculated      *big.Int // The additive amount of total output\input calculated trough the swap
	// totalFeeGrowth                *big.Int // The initial totalFeeGrowth + the fee growth during a swap
	// totalFeeGrowthB               *big.Int
	// incentiveStatus               IAlgebraVirtualPool.Status // If there is an active incentive at the moment
	exactInput bool   // Whether the exact input or output is specified
	fee        uint16 // The current dynamic fee
	// startTick  int    // The tick at the start of a swap
	// timepointIndex uint16 // The index of last written timepoint
}

type PriceMovementCache struct {
	stepSqrtPrice *big.Int // The Q64.96 sqrt of the price at the start of the step
	nextTick      int      // The tick till the current step goes
	initialized   bool     // True if the _nextTick is initialized
	nextTickPrice *big.Int // The Q64.96 sqrt of the price calculated from the _nextTick
	input         *big.Int // The additive amount of tokens that have been provided
	output        *big.Int // The additive amount of token that have been withdrawn
	feeAmount     *big.Int // The total amount of fee earned within a current step
}

// https://github.com/cryptoalgebra/AlgebraV1/blob/dfebf532a27803dafcbf2ba49724740bd6220505/src/core/contracts/AlgebraPool.sol#L703
func (p *PoolSimulator) _calculateSwapAndLock(
	zeroToOne bool,
	amountRequired *big.Int,
	limitSqrtPrice *big.Int,
) (error, *big.Int, *big.Int, *StateUpdate) {
	var cache SwapCalculationCache
	var err error

	nextState := &StateUpdate{}

	// load from one storage slot
	currentPrice := p.globalState.Price
	currentTick := int(p.globalState.Tick.Int64())
	cache.amountCalculated = integer.Zero()
	_communityFeeToken0 := p.globalState.CommunityFeeToken0
	_communityFeeToken1 := p.globalState.CommunityFeeToken1

	cmp := amountRequired.Cmp(integer.Zero())
	if cmp == 0 {
		return ErrZeroAmountIn, nil, nil, nil
	}

	cache.amountRequiredInitial, cache.exactInput = amountRequired, cmp > 0

	currentLiquidity := p.liquidity

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(utils.MinSqrtRatio) <= 0 {
			return ErrSPL, nil, nil, nil
		}
		cache.communityFee = big.NewInt(int64(_communityFeeToken0))
	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(utils.MaxSqrtRatio) >= 0 {
			return ErrSPL, nil, nil, nil
		}
		cache.communityFee = big.NewInt(int64(_communityFeeToken1))
	}

	// don't need to care about activeIncentive

	// use pre-calculated fee instead of calculating from timepoints
	// see tracker code for more details
	if zeroToOne {
		cache.fee = p.globalState.FeeZto
	} else {
		cache.fee = p.globalState.FeeOtz
	}
	logger.Debugf("fee %v", cache.fee)

	var step PriceMovementCache
	// swap until there is remaining input or output tokens or we reach the price limit
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		step.stepSqrtPrice = currentPrice

		step.nextTick, step.initialized, err = p.ticks.NextInitializedTickWithinOneWord(currentTick, zeroToOne, p.tickSpacing)
		if err != nil {
			return err, nil, nil, nil
		}

		step.nextTickPrice, err = utils.GetSqrtRatioAtTick(step.nextTick)
		if err != nil {
			return err, nil, nil, nil
		}

		// calculate the amounts needed to move the price to the next target if it is possible or as much as possible
		targetPrice := step.nextTickPrice
		ltLimit := step.nextTickPrice.Cmp(limitSqrtPrice) < 0
		if zeroToOne == ltLimit {
			targetPrice = limitSqrtPrice
		}
		currentPrice, step.input, step.output, step.feeAmount, err = utils.ComputeSwapStep(
			currentPrice,
			targetPrice,
			currentLiquidity,
			amountRequired,
			constants.FeeAmount(cache.fee),
		)
		if err != nil {
			return err, nil, nil, nil
		}

		if cache.exactInput {
			amountRequired = new(big.Int).Sub(amountRequired, new(big.Int).Add(step.input, step.feeAmount)) // decrease remaining input amount
			cache.amountCalculated = new(big.Int).Sub(cache.amountCalculated, step.output)                  // decrease calculated output amount
		} else {
			amountRequired = new(big.Int).Add(amountRequired, step.output) // increase remaining output amount (since its negative)
			cache.amountCalculated = new(big.Int).Add(cache.amountCalculated,
				new(big.Int).Add(step.input, step.feeAmount),
			) // increase calculated input amount
		}

		if cache.communityFee.Cmp(integer.Zero()) > 0 {
			delta := new(big.Int).Div(
				new(big.Int).Mul(step.feeAmount, cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)
			step.feeAmount = new(big.Int).Sub(step.feeAmount, delta)
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
					return err, nil, nil, nil
				}
				var liquidityDelta *big.Int
				if zeroToOne {
					liquidityDelta = new(big.Int).Neg(nextTickData.LiquidityNet)
				} else {
					liquidityDelta = nextTickData.LiquidityNet
				}

				currentLiquidity = utils.AddDelta(currentLiquidity, liquidityDelta)
			}
			if zeroToOne {
				currentTick = step.nextTick - 1
			} else {
				currentTick = step.nextTick
			}
		} else if currentPrice != step.stepSqrtPrice {
			// if the price has changed but hasn't reached the target
			currentTick, err = utils.GetTickAtSqrtRatio(currentPrice)
			if err != nil {
				return err, nil, nil, nil
			}
			break // since the price hasn't reached the target, amountRequired should be 0
		}

		// check stop condition
		if amountRequired.Cmp(integer.Zero()) == 0 || currentPrice.Cmp(limitSqrtPrice) == 0 {
			break
		}
	}

	var amount0, amount1 *big.Int
	// the amount to provide could be less then initially specified (e.g. reached limit)
	if zeroToOne == cache.exactInput {
		// the amount to get could be less then initially specified (e.g. reached limit)
		amount0, amount1 = new(big.Int).Sub(cache.amountRequiredInitial, amountRequired), cache.amountCalculated
	} else {
		amount0, amount1 = cache.amountCalculated, new(big.Int).Sub(cache.amountRequiredInitial, amountRequired)
	}

	nextState.GlobalState = GlobalState{
		Price:              currentPrice,
		Tick:               big.NewInt(int64(currentTick)),
		FeeZto:             p.globalState.FeeZto,
		FeeOtz:             p.globalState.FeeOtz,
		TimepointIndex:     p.globalState.TimepointIndex,
		CommunityFeeToken0: p.globalState.CommunityFeeToken0,
		CommunityFeeToken1: p.globalState.CommunityFeeToken0,
	}

	nextState.Liquidity = currentLiquidity

	return nil, amount0, amount1, nextState
}
