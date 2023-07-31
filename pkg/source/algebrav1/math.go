package algebrav1

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
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
	// totalFeeGrowth                *big.Int // The initial totalFeeGrowth + the fee growth during a swap
	// totalFeeGrowthB               *big.Int
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

func (p *PoolSimulator) _getSingleTimepoint(
	blockTimestamp uint32,
	secondsAgo uint32,
	startTick int24,
	timepointIndex uint16,
	liquidityStart *big.Int,
) (error, int56, *big.Int, *big.Int, *big.Int) {

	var oldestIndex uint16
	// check if we have overflow in the past
	nextIndex := timepointIndex + 1 // considering overflow
	if p.timepoints.Get(nextIndex).Initialized {
		oldestIndex = nextIndex
	}

	result, err := p.timepoints.getSingleTimepoint(blockTimestamp, secondsAgo, startTick, timepointIndex, oldestIndex, liquidityStart)
	if err != nil {
		return err, 0, nil, nil, nil
	}
	return nil,
		result.TickCumulative,
		result.SecondsPerLiquidityCumulative,
		result.VolatilityCumulative,
		result.VolumePerLiquidityCumulative
}

func (p *PoolSimulator) _calculateSwapAndLock(
	zeroToOne bool,
	amountRequired *big.Int,
	limitSqrtPrice *big.Int,
) (error, *big.Int, *big.Int, *StateUpdate) {
	nextState := &StateUpdate{}

	defer func() {
		// reset written timepoints
		p.timepoints.updates = map[uint16]Timepoint{}
	}()

	// blockTimestamp := uint32(time.Now().Unix())
	blockTimestamp := uint32(1690776535)
	var cache SwapCalculationCache

	// load from one storage slot
	currentPrice := p.globalState.Price
	currentTick := int(p.globalState.Tick.Int64())
	cache.fee = p.globalState.Fee
	cache.amountCalculated = bignumber.ZeroBI
	cache.timepointIndex = p.globalState.TimepointIndex
	_communityFeeToken0 := p.globalState.CommunityFeeToken0
	_communityFeeToken1 := p.globalState.CommunityFeeToken1

	communityFeeAmount := bignumber.ZeroBI // TODO: return this

	cmp := amountRequired.Cmp(bignumber.ZeroBI)
	if cmp == 0 {
		return errors.New("AS"), nil, nil, nil
	}

	cache.amountRequiredInitial, cache.exactInput = amountRequired, cmp > 0

	var currentLiquidity *big.Int
	// currentLiquidity, cache.volumePerLiquidityInBlock = p.liquidity, p.volumePerLiquidityInBlock
	currentLiquidity, cache.volumePerLiquidityInBlock = p.liquidity, bignumber.NewBig10("172760224274117266")

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(utils.MinSqrtRatio) <= 0 {
			return errors.New("SPL"), nil, nil, nil
		}
		cache.communityFee = big.NewInt(int64(_communityFeeToken0))
	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(utils.MaxSqrtRatio) >= 0 {
			return errors.New("SPL"), nil, nil, nil
		}
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
		return err, nil, nil, nil
	}

	// new timepoint appears only for first swap in block
	if newTimepointIndex != cache.timepointIndex {
		cache.timepointIndex = newTimepointIndex
		cache.volumePerLiquidityInBlock = bignumber.ZeroBI
		cache.fee, err = p._getNewFee(blockTimestamp, int24(currentTick), newTimepointIndex, currentLiquidity)
		logger.Debugf("fee %v", cache.fee)
		if err != nil {
			return err, nil, nil, nil
		}
	}

	var step PriceMovementCache
	// swap until there is remaining input or output tokens or we reach the price limit
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		step.stepSqrtPrice = currentPrice

		step.nextTick, step.initialized = p.ticks.NextInitializedTickWithinOneWord(currentTick, zeroToOne, p.tickSpacing)

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

		if cache.communityFee.Cmp(bignumber.ZeroBI) > 0 {
			delta := new(big.Int).Div(
				new(big.Int).Mul(step.feeAmount, cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)
			step.feeAmount = new(big.Int).Sub(step.feeAmount, delta)
			communityFeeAmount = new(big.Int).Add(communityFeeAmount, delta)
		}

		if currentPrice == step.nextTickPrice {
			// if the reached tick is initialized then we need to cross it
			if step.initialized {
				// once at a swap we have to get the last timepoint of the observation
				if !cache.computedLatestTimepoint {
					err, cache.tickCumulative, cache.secondsPerLiquidityCumulative, _, _ = p._getSingleTimepoint(
						blockTimestamp,
						0,
						int24(cache.startTick),
						cache.timepointIndex,
						currentLiquidity, // currentLiquidity can be changed only after computedLatestTimepoint
					)
					if err != nil {
						return err, nil, nil, nil
					}
					cache.computedLatestTimepoint = true
				}

				// every tick cross is needed to be duplicated in a virtual pool
				// don't need to do this here

				var liquidityDelta *big.Int
				if zeroToOne {
					liquidityDelta = new(big.Int).Neg(p.ticks.GetTick(step.nextTick).LiquidityNet)
				} else {
					liquidityDelta = p.ticks.GetTick(step.nextTick).LiquidityNet
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

	nextState.GlobalState = GlobalState{
		Price:          currentPrice,
		Tick:           big.NewInt(int64(currentTick)),
		Fee:            cache.fee,
		TimepointIndex: cache.timepointIndex,
	}

	nextState.Liquidity, nextState.VolumePerLiquidityInBlock =
		currentLiquidity,
		new(big.Int).Add(cache.volumePerLiquidityInBlock, calculateVolumePerLiquidity(currentLiquidity, amount0, amount1))

	// copy written timepoints
	nextState.NewTimepoints = make(map[uint16]Timepoint, len(p.timepoints.updates))
	for i, tp := range p.timepoints.updates {
		nextState.NewTimepoints[i] = tp
	}

	fmt.Println("amount--", amount0, amount1)
	return nil, amount0, amount1, nextState
}

// func MulDivRoundingDown(a, b, denominator *big.Int) *big.Int {
// 	product := new(big.Int).Mul(a, b)
// 	result := new(big.Int).Div(product, denominator)
// 	return result
// }

// / @notice Transitions to next tick as needed by price movement
// / @param self The mapping containing all tick information for initialized ticks
// / @param tick The destination tick of the transition
// / @param totalFeeGrowth0Token The all-time global fee growth, per unit of liquidity, in token0
// / @param totalFeeGrowth1Token The all-time global fee growth, per unit of liquidity, in token1
// / @param secondsPerLiquidityCumulative The current seconds per liquidity
// / @param tickCumulative The all-time global cumulative tick
// / @param time The current block.timestamp
// / @return liquidityDelta The amount of liquidity added (subtracted) when tick is crossed from left to right (right to left)
// func cross(
// 	ticks *entities.TickListDataProvider,
// 	tick int24,
// 	totalFeeGrowth0Token *big.Int,
// 	totalFeeGrowth1Token *big.Int,
// 	secondsPerLiquidityCumulative *big.Int,
// 	tickCumulative int56,
// 	time uint32,
// ) *big.Int {
// 	data := ticks.GetTick(int(tick))

// 	data.outerSecondsSpent = time - data.outerSecondsSpent
// 	data.outerSecondsPerLiquidity = secondsPerLiquidityCumulative - data.outerSecondsPerLiquidity
// 	data.outerTickCumulative = tickCumulative - data.outerTickCumulative

// 	data.outerFeeGrowth1Token = totalFeeGrowth1Token - data.outerFeeGrowth1Token
// 	data.outerFeeGrowth0Token = totalFeeGrowth0Token - data.outerFeeGrowth0Token

// 	return data.liquidityDelta
// }

func calculateVolumePerLiquidity(
	liquidity *big.Int,
	amount0 *big.Int,
	amount1 *big.Int,
) *big.Int {
	volume := new(big.Int).Mul(sqrtAbs(amount0), sqrtAbs(amount1))
	var volumeShifted *big.Int
	if liquidity.Cmp(bignumber.ZeroBI) <= 0 {
		liquidity = new(big.Int).Set(bignumber.One)
	}
	if volume.Cmp(pow192) >= 0 {
		volumeShifted = new(big.Int).Div(uint256_max, liquidity)
	} else {
		volumeShifted = new(big.Int).Div(new(big.Int).Lsh(volume, 64), liquidity)
	}
	if volumeShifted.Cmp(MAX_VOLUME_PER_LIQUIDITY) >= 0 {
		return MAX_VOLUME_PER_LIQUIDITY
	} else {
		return volumeShifted
	}
}

// / @notice Gets the square root of the absolute value of the parameter
func sqrtAbs(_x *big.Int) *big.Int {
	// get abs value
	mask := new(big.Int).Rsh(_x, (256 - 1))
	x := new(big.Int).Sub(new(big.Int).Xor(_x, mask), mask)
	if x.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	} else {
		xx := x
		r := new(big.Int).Set(bignumber.One)
		if xx.Cmp(bignumber.NewBig("0x100000000000000000000000000000000")) >= 0 {
			xx = new(big.Int).Rsh(xx, 128)
			r = new(big.Int).Lsh(r, 64)
		}
		if xx.Cmp(bignumber.NewBig("0x10000000000000000")) >= 0 {
			xx = new(big.Int).Rsh(xx, 64)
			r = new(big.Int).Lsh(r, 32)
		}
		if xx.Cmp(bignumber.NewBig("0x100000000")) >= 0 {
			xx = new(big.Int).Rsh(xx, 32)
			r = new(big.Int).Lsh(r, 16)
		}
		if xx.Cmp(bignumber.NewBig("0x10000")) >= 0 {
			xx = new(big.Int).Rsh(xx, 16)
			r = new(big.Int).Lsh(r, 8)
		}
		if xx.Cmp(bignumber.NewBig("0x100")) >= 0 {
			xx = new(big.Int).Rsh(xx, 8)
			r = new(big.Int).Lsh(r, 4)
		}
		if xx.Cmp(bignumber.NewBig("0x10")) >= 0 {
			xx = new(big.Int).Rsh(xx, 4)
			r = new(big.Int).Lsh(r, 2)
		}
		if xx.Cmp(bignumber.NewBig("0x8")) >= 0 {
			r = new(big.Int).Lsh(r, 1)
		}
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1)
		r = new(big.Int).Rsh(new(big.Int).Add(r, new(big.Int).Div(x, r)), 1) // @dev Seven iterations should be enough.
		r1 := new(big.Int).Div(x, r)
		if r.Cmp(r1) < 0 {
			return r
		}
		return r1
	}
}
