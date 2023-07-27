package algebrav1

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/daoleno/uniswapv3-sdk/utils"
)

type SwapCalculationCache struct {
	communityFee                  *big.Int // The community fee of the selling token, uint256 to minimize casts
	volumePerLiquidityInBlock     *big.Int
	tickCumulative                int64    // The global tickCumulative at the moment
	secondsPerLiquidityCumulative *big.Int // The global secondPerLiquidity at the moment
	computedLatestTimepoint       bool     //  if we have already fetched _tickCumulative_ and _secondPerLiquidity_ from the DataOperator
	amountRequiredInitial         *big.Int // The initial value of the exact input\output amount
	amountCalculated              *big.Int // The additive amount of total output\input calculated trough the swap
	totalFeeGrowth                *big.Int // The initial totalFeeGrowth + the fee growth during a swap
	totalFeeGrowthB               *big.Int
	// incentiveStatus               IAlgebraVirtualPool.Status // If there is an active incentive at the moment
	exactInput     bool   // Whether the exact input or output is specified
	fee            uint16 // The current dynamic fee
	startTick      int    // The tick at the start of a swap
	timepointIndex uint16 // The index of last written timepoint
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

func _require(cond bool, msg string) {
	if cond {
		panic(msg)
	}
}

func (p *PoolSimulator) _writeTimepoint(
	timepointIndex uint16,
	blockTimestamp uint32,
	tick int24,
	liquidity *big.Int,
	volumePerLiquidityInBlock *big.Int,
) (uint16, error) {
	return p.timepoints.write(timepointIndex, blockTimestamp, tick, liquidity, volumePerLiquidityInBlock)
}

func (p *PoolSimulator) _getNewFee(
	_time uint32,
	_tick int24,
	_index uint16,
	_liquidity *big.Int,
) (uint16, error) {
	err, volatilityAverage, volumePerLiqAverage := p.timepoints.getAverages(_time, _tick, _index, _liquidity)
	if err != nil {
		return 0, err
	}
	return getFee(
		new(big.Int).Div(volatilityAverage, big.NewInt(15)),
		volumePerLiqAverage,
		&p.feeConf,
	), nil
}

func (p *PoolSimulator) _calculateSwapAndLock(
	zeroToOne bool,
	amountRequired *big.Int,
	limitSqrtPrice *big.Int,
) (error, *big.Int, *big.Int) {

	blockTimestamp := uint32(time.Now().Unix())
	var cache SwapCalculationCache

	// load from one storage slot
	currentPrice := p.globalState.Price
	currentTick := int(p.globalState.Tick.Int64()) // TODO: cast at tracker side
	cache.fee = p.globalState.Fee
	cache.amountCalculated = bignumber.ZeroBI
	cache.timepointIndex = p.globalState.TimepointIndex
	_communityFeeToken0 := p.globalState.CommunityFeeToken0
	_communityFeeToken1 := p.globalState.CommunityFeeToken1

	communityFeeAmount := bignumber.ZeroBI // TODO: return this

	cmp := amountRequired.Cmp(bignumber.ZeroBI)
	if cmp == 0 {
		return errors.New("AS"), nil, nil
	}

	cache.amountRequiredInitial, cache.exactInput = amountRequired, cmp > 0

	var currentLiquidity *big.Int
	currentLiquidity, cache.volumePerLiquidityInBlock = p.liquidity, p.volumePerLiquidityInBlock

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(utils.MinSqrtRatio) <= 0 {
			return errors.New("SPL"), nil, nil
		}
		cache.totalFeeGrowth = p.totalFeeGrowth0Token
		cache.communityFee = big.NewInt(int64(_communityFeeToken0))
	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(utils.MaxSqrtRatio) >= 0 {
			return errors.New("SPL"), nil, nil
		}
		cache.totalFeeGrowth = p.totalFeeGrowth1Token
		cache.communityFee = big.NewInt(int64(_communityFeeToken1))
	}

	cache.startTick = currentTick

	// don't need to care about activeIncentive

	newTimepointIndex, err := p._writeTimepoint(
		cache.timepointIndex,
		blockTimestamp,
		int24(cache.startTick),
		currentLiquidity,
		cache.volumePerLiquidityInBlock,
	)
	if err != nil {
		return err, nil, nil
	}

	// new timepoint appears only for first swap in block
	if newTimepointIndex != cache.timepointIndex {
		cache.timepointIndex = newTimepointIndex
		cache.volumePerLiquidityInBlock = bignumber.ZeroBI
		cache.fee, err = p._getNewFee(blockTimestamp, int24(currentTick), newTimepointIndex, currentLiquidity)
		if err != nil {
			return err, nil, nil
		}
	}

	var step PriceMovementCache
	// swap until there is remaining input or output tokens or we reach the price limit
	for {
		step.stepSqrtPrice = currentPrice

		step.nextTick, step.initialized = p.ticks.NextInitializedTickWithinOneWord(currentTick, zeroToOne, tickSpacing)

		step.nextTickPrice, err = utils.GetSqrtRatioAtTick(step.nextTick)
		if err != nil {
			return err, nil, nil
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
			return err, nil, nil
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

		if cache.communityFee.Cmp(bignumber.ZeroBI) > 0 {
			delta := new(big.Int).Div(
				new(big.Int).Mul(step.feeAmount, cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)
			step.feeAmount = new(big.Int).Sub(step.feeAmount, delta)
			communityFeeAmount = new(big.Int).Add(communityFeeAmount, delta)
		}

		if currentLiquidity.Cmp(bignumber.ZeroBI) > 0 {
			cache.totalFeeGrowth = new(big.Int).Add(cache.totalFeeGrowth, MulDivRoundingDown(step.feeAmount, Q128, currentLiquidity))
		}

		if currentPrice == step.nextTickPrice {
			// if the reached tick is initialized then we need to cross it
			if step.initialized {
				// once at a swap we have to get the last timepoint of the observation
				if !cache.computedLatestTimepoint {
					// TODO
					// cache.tickCumulative, cache.secondsPerLiquidityCumulative, _, _ = _getSingleTimepoint(
					// 	blockTimestamp,
					// 	0,
					// 	cache.startTick,
					// 	cache.timepointIndex,
					// 	currentLiquidity, // currentLiquidity can be changed only after computedLatestTimepoint
					// )
					// cache.computedLatestTimepoint = true
					// if zeroToOne {
					// 	cache.totalFeeGrowthB = p.totalFeeGrowth1Token
					// } else {
					// 	cache.totalFeeGrowthB = p.totalFeeGrowth0Token
					// }
				}

				// every tick cross is needed to be duplicated in a virtual pool
				// don't need to do this here

				// TODO
				// var liquidityDelta *big.Int
				// if zeroToOne {
				// 	liquidityDelta = -ticks.cross(
				// 		step.nextTick,
				// 		cache.totalFeeGrowth,  // A == 0
				// 		cache.totalFeeGrowthB, // B == 1
				// 		cache.secondsPerLiquidityCumulative,
				// 		cache.tickCumulative,
				// 		blockTimestamp,
				// 	)
				// } else {
				// 	liquidityDelta = ticks.cross(
				// 		step.nextTick,
				// 		cache.totalFeeGrowthB, // B == 0
				// 		cache.totalFeeGrowth,  // A == 1
				// 		cache.secondsPerLiquidityCumulative,
				// 		cache.tickCumulative,
				// 		blockTimestamp,
				// 	)
				// }

				// currentLiquidity = LiquidityMath.addDelta(currentLiquidity, liquidityDelta)
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
				return err, nil, nil
			}
			break // since the price hasn't reached the target, amountRequired should be 0
		}

		// check stop condition
		if amountRequired.Cmp(bignumber.ZeroBI) == 0 || currentPrice.Cmp(limitSqrtPrice) == 0 {
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
	// TODO: do not update here but return to be updated later
	// p.globalState.Price, p.globalState.Tick, p.globalState.Fee, p.globalState.TimepointIndex = currentPrice, currentTick, cache.fee, cache.timepointIndex

	// liquidity, volumePerLiquidityInBlock =
	// 	currentLiquidity,
	// 	cache.volumePerLiquidityInBlock+IDataStorageOperator(dataStorageOperator).calculateVolumePerLiquidity(currentLiquidity, amount0, amount1)

	// if zeroToOne {
	// 	totalFeeGrowth0Token = cache.totalFeeGrowth
	// } else {
	// 	totalFeeGrowth1Token = cache.totalFeeGrowth
	// }

	fmt.Println("amount--", amount0, amount1)
	return nil, amount0, amount1
}

func MulDivRoundingDown(a, b, denominator *big.Int) *big.Int {
	product := new(big.Int).Mul(a, b)
	result := new(big.Int).Div(product, denominator)
	return result
}
