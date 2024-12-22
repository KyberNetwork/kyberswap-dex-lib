package integral

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	globalState GlobalState
	liquidity   *uint256.Int

	ticks       *v3Entities.TickListDataProvider
	gas         int64
	tickMin     int32
	tickMax     int32
	tickSpacing int

	timepoints         *TimepointStorage
	volatilityOracle   *VolatilityOraclePlugin
	dynamicFee         *DynamicFeeConfig
	slidingFee         *SlidingFeeConfig
	writeTimePointOnce *sync.Once

	useBasePluginV2 bool
}

func NewPoolSimulator(entityPool entity.Pool, defaultGas int64) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	} else {
		return nil, ErrInvalidToken
	}

	// if the tick list is empty, the pool should be ignored
	if len(extra.Ticks) == 0 {
		return nil, ErrTicksEmpty
	}

	ticks, err := v3Entities.NewTickListDataProvider(extra.Ticks, int(extra.TickSpacing))
	if err != nil {
		return nil, err
	}

	timepoints := NewTimepointStorage(extra.Timepoints)

	tickMin := extra.Ticks[0].Index
	tickMax := extra.Ticks[len(extra.Ticks)-1].Index

	var info = pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		ReserveUsd:  entityPool.ReserveUsd,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:               pool.Pool{Info: info},
		globalState:        extra.GlobalState,
		liquidity:          uint256.MustFromBig(extra.Liquidity),
		ticks:              ticks,
		gas:                defaultGas,
		tickMin:            int32(tickMin),
		tickMax:            int32(tickMax),
		tickSpacing:        int(extra.TickSpacing),
		timepoints:         timepoints,
		volatilityOracle:   &extra.VolatilityOracle,
		dynamicFee:         &extra.DynamicFee,
		slidingFee:         &extra.SlidingFee,
		writeTimePointOnce: new(sync.Once),
		useBasePluginV2:    staticExtra.UseBasePluginV2,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var (
		tokenInIndex  = p.GetTokenIndex(tokenAmountIn.Token)
		tokenOutIndex = p.GetTokenIndex(tokenOut)

		zeroForOne             bool
		overrideFee, pluginFee uint32
		err                    error
	)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	if tokenInIndex == 0 {
		zeroForOne = true
	}

	if p.useBasePluginV2 {
		overrideFee, pluginFee, err = p.beforeSwapV2(zeroForOne)
		if err != nil {
			return nil, err
		}
	} else {
		overrideFee, pluginFee, err = p.beforeSwapV1()
		if err != nil {
			return nil, err
		}
	}

	if !p.globalState.Unlocked {
		return nil, errors.New("pool has been locked and not usable")
	}

	priceLimit, err := p.getSqrtPriceLimit(zeroForOne)
	if err != nil {
		return nil, err
	}

	amountRequired, err := int256.FromBig(tokenAmountIn.Amount)
	if err != nil {
		return nil, ErrInvalidAmountRequired
	}

	amount0, amount1, currentPrice, currentTick, currentLiquidity, fees, err := p.calculateSwap(
		overrideFee, pluginFee, zeroForOne, amountRequired, priceLimit)
	if err != nil {
		return nil, err
	}

	newState := GlobalState{
		Price:        currentPrice,
		Tick:         currentTick,
		LastFee:      p.globalState.LastFee,
		PluginConfig: p.globalState.PluginConfig,
		CommunityFee: p.globalState.CommunityFee,
		Unlocked:     p.globalState.Unlocked,
	}

	var amountOut *int256.Int
	if zeroForOne {
		amountOut = new(int256.Int).Neg(amount1)
	} else {
		amountOut = new(int256.Int).Neg(amount0)
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenAmountIn.Token,
			Amount: new(uint256.Int).Add(fees.communityFeeAmount, fees.pluginFeeAmount).ToBig(),
		},
		Gas: p.gas,
		SwapInfo: StateUpdate{
			GlobalState: newState,
			Liquidity:   currentLiquidity,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.liquidity = p.liquidity.Clone()
	cloned.globalState.Price = p.globalState.Price.Clone()
	cloned.writeTimePointOnce = new(sync.Once)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(StateUpdate)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Algebra %v %v pool, wrong swapInfo type",
			p.Info.Address, p.Info.Exchange)
		return
	}
	p.liquidity = new(uint256.Int).Set(si.Liquidity)
	p.globalState = si.GlobalState
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	zeroForOne := strings.EqualFold(tokenIn, p.Info.Tokens[0])
	priceLimit, _ := p.getSqrtPriceLimit(zeroForOne)
	return PoolMeta{
		BlockNumber: p.Pool.Info.BlockNumber,
		PriceLimit:  priceLimit,
	}
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool) (*uint256.Int, error) {
	var tickLimit int32
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	var sqrtPriceX96Limit v3Utils.Uint160
	err := v3Utils.GetSqrtRatioAtTickV2(int(tickLimit), &sqrtPriceX96Limit)
	if err != nil {
		return nil, err
	}

	if zeroForOne {
		sqrtPriceX96Limit.AddUint64(&sqrtPriceX96Limit, 1) // = (sqrtPrice at minTick) + 1
	} else {
		sqrtPriceX96Limit.SubUint64(&sqrtPriceX96Limit, 1) // = (sqrtPrice at maxTick) - 1
	}

	return &sqrtPriceX96Limit, nil
}

// writeTimepoint locks and writes timepoint only once, triggering onWrite only if said write happened.
// By right we should re-update the timepoint every new second, but the difference should be small enough, and
// new pool should have already been created and used in replacement of this pool.
func (p *PoolSimulator) writeTimepoint(onWrite func() error) (err error) {
	volatilityOracle := p.volatilityOracle
	if !volatilityOracle.IsInitialized {
		return ErrNotInitialized
	}

	p.writeTimePointOnce.Do(func() {
		volatilityOracle.LastTimepointTimestamp = uint32(time.Now().Unix())
		volatilityOracle.TimepointIndex, _, err = p.timepoints.write(
			volatilityOracle.TimepointIndex, volatilityOracle.LastTimepointTimestamp, p.globalState.Tick)
		if err != nil || onWrite == nil {
			return
		}
		err = onWrite()
	})
	return err
}

func (p *PoolSimulator) beforeSwapV1() (uint32, uint32, error) {
	if p.globalState.PluginConfig&BEFORE_SWAP_FLAG == 0 {
		return 0, 0, nil
	}
	return 0, 0, p.writeTimepoint(func() error {
		volatilityLast, err := p.getAverageVolatilityLast()
		if err != nil {
			return err
		}
		var newFee uint16
		if p.dynamicFee.Alpha1 == 0 && p.dynamicFee.Alpha2 == 0 {
			newFee = p.dynamicFee.BaseFee
		} else {
			newFee = getFee(volatilityLast, p.dynamicFee)
		}
		p.globalState.LastFee = newFee
		return nil
	})
}

func (p *PoolSimulator) beforeSwapV2(zeroToOne bool) (uint32, uint32, error) {
	currentTick := p.globalState.Tick
	lastTick := p.getLastTick()

	newFee, err := p.getFeeAndUpdateFactors(zeroToOne, currentTick, lastTick)
	if err != nil {
		return 0, 0, err
	}

	if err := p.writeTimepoint(nil); err != nil {
		return 0, 0, err
	}

	return uint32(newFee), 0, nil
}

func (p *PoolSimulator) getFeeAndUpdateFactors(zeroToOne bool, currentTick, lastTick int32) (uint16, error) {
	var currentFeeFactors *SlidingFeeConfig

	if currentTick != lastTick {
		var err error
		currentFeeFactors, err = calculateFeeFactors(currentTick, lastTick, s_priceChangeFactor)
		if err != nil {
			return 0, err
		}

		p.slidingFee = currentFeeFactors
	} else {
		currentFeeFactors = p.slidingFee
	}

	var adjustedFee *uint256.Int
	baseFeeBig := uint256.NewInt(s_baseFee)
	adjustedFee = baseFeeBig.Rsh(
		baseFeeBig.Mul(baseFeeBig,
			lo.Ternary(zeroToOne, currentFeeFactors.ZeroToOneFeeFactor, currentFeeFactors.OneToZeroFeeFactor),
		),
		FEE_FACTOR_SHIFT,
	)

	if adjustedFee.Cmp(MAX_UINT16) > 0 || adjustedFee.IsZero() {
		adjustedFee.Set(MAX_UINT16)
	}

	return uint16(adjustedFee.Uint64()), nil
}

func (p *PoolSimulator) getLastTick() int32 {
	lastTimepointIndex := p.volatilityOracle.TimepointIndex
	lastTimepoint := p.timepoints.Get(lastTimepointIndex)
	return lastTimepoint.Tick
}

func (p *PoolSimulator) getAverageVolatilityLast() (*uint256.Int, error) {
	currentTimestamp := p.volatilityOracle.LastTimepointTimestamp
	tick := p.globalState.Tick
	lastTimepointIndex := p.volatilityOracle.TimepointIndex
	oldestIndex := p.timepoints.getOldestIndex(lastTimepointIndex)

	volatilityAverage, err := p.timepoints.getAverageVolatility(currentTimestamp, tick, lastTimepointIndex, oldestIndex)
	if err != nil {
		return nil, err
	}

	return volatilityAverage, nil
}

func (p *PoolSimulator) calculateSwap(overrideFee, pluginFee uint32, zeroToOne bool, amountRequired *int256.Int,
	limitSqrtPrice *uint256.Int) (*int256.Int, *int256.Int, *uint256.Int, int32, *uint256.Int, FeesAmount, error) {
	if amountRequired.IsZero() {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrZeroAmountRequired
	}

	var cache SwapCalculationCache
	var fees = FeesAmount{
		communityFeeAmount: new(uint256.Int),
		pluginFeeAmount:    new(uint256.Int),
	}

	cache.amountRequiredInitial = amountRequired
	cache.exactInput = amountRequired.IsPositive()
	cache.pluginFee = pluginFee
	cache.amountCalculated = new(int256.Int)

	currentLiquidity := p.liquidity

	currentPrice := p.globalState.Price
	currentTick := p.globalState.Tick
	cache.fee = uint32(p.globalState.LastFee)
	cache.communityFee = uint256.NewInt(uint64(p.globalState.CommunityFee))

	if currentPrice.IsZero() {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrNotInitialized
	}

	if overrideFee != 0 {
		cache.fee = overrideFee + pluginFee
	} else {
		if pluginFee != 0 {
			cache.fee += pluginFee
		}

		if cache.fee >= 1e6 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrIncorrectPluginFee
		}
	}

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(uint256.MustFromBig(MIN_SQRT_RATIO)) <= 0 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidLimitSqrtPrice
		}
	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(uint256.MustFromBig(MAX_SQRT_RATIO)) >= 0 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidLimitSqrtPrice
		}
	}

	var step PriceMovementCache
	initializedTick := currentTick
	// swap until there is remaining input or output tokens, or we reach the price limit.
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		var (
			nextTick int
			err      error
		)

		nextTick, step.initialized, err = p.ticks.NextInitializedTickWithinOneWord(int(initializedTick), zeroToOne,
			p.tickSpacing)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}
		step.nextTick = int32(nextTick)

		if !step.initialized {
			if zeroToOne {
				initializedTick = step.nextTick - 1
			} else {
				initializedTick = step.nextTick
			}
			continue
		}

		step.stepSqrtPrice = currentPrice

		nextTickPrice, err := utils.GetSqrtRatioAtTick(int(step.nextTick))
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}
		step.nextTickPrice = uint256.MustFromBig(nextTickPrice)

		var targetPrice = step.nextTickPrice
		if zeroToOne == (step.nextTickPrice.Cmp(limitSqrtPrice) < 0) {
			targetPrice = limitSqrtPrice
		}

		currentPrice, step.input, step.output, step.feeAmount, err = movePriceTowardsTarget(zeroToOne, currentPrice,
			targetPrice, currentLiquidity, amountRequired, cache.fee)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		output, err := ToInt256(step.output)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		amountDelta, err := ToInt256(new(uint256.Int).Add(step.input, step.feeAmount))
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		if cache.exactInput {
			amountRequired.Sub(amountRequired, amountDelta)
			cache.amountCalculated = new(int256.Int).Sub(cache.amountCalculated, output)
		} else {
			amountRequired.Add(amountRequired, output)
			cache.amountCalculated = new(int256.Int).Add(cache.amountCalculated, amountDelta)
		}

		if cache.pluginFee > 0 && cache.fee > 0 {
			delta, err := v3Utils.MulDiv(step.feeAmount, uint256.NewInt(uint64(cache.pluginFee)),
				uint256.NewInt(uint64(cache.fee)))
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.pluginFeeAmount.Add(fees.pluginFeeAmount, delta)
		}

		if cache.communityFee.Sign() > 0 {
			delta := new(uint256.Int).Div(
				new(uint256.Int).Mul(step.feeAmount, cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.communityFeeAmount.Add(fees.communityFeeAmount, delta)
		}

		if currentPrice.Cmp(step.nextTickPrice) == 0 {
			tickData, err := p.ticks.GetTick(int(step.nextTick))
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			liquidityNet := int256.MustFromBig(tickData.LiquidityNet)

			var liquidityDelta = new(int256.Int)
			if zeroToOne {
				currentTick = step.nextTick - 1
				initializedTick = step.nextTick - 1
				liquidityDelta = liquidityDelta.Neg(liquidityNet)
			} else {
				currentTick = step.nextTick
				initializedTick = step.nextTick
				liquidityDelta = liquidityNet
			}

			currentLiquidity, err = addDelta(currentLiquidity, liquidityDelta)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

		} else if currentPrice.Cmp(step.stepSqrtPrice) != 0 {
			currentTickInt, err := utils.GetTickAtSqrtRatio(currentPrice.ToBig())
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			currentTick = int32(currentTickInt)

			break
		}

		if amountRequired.IsZero() || currentPrice.Cmp(limitSqrtPrice) == 0 {
			break
		}
	}

	amountSpent := new(int256.Int).Sub(cache.amountRequiredInitial, amountRequired)

	amount0, amount1 := cache.amountCalculated, amountSpent
	if zeroToOne == cache.exactInput {
		amount0, amount1 = amountSpent, cache.amountCalculated
	}

	return amount0, amount1, currentPrice, currentTick, currentLiquidity, fees, nil
}

func movePriceTowardsTarget(
	zeroToOne bool,
	currentPrice, targetPrice, liquidity *uint256.Int,
	amountAvailableInt256 *int256.Int,
	fee uint32,
) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	amountAvailable, err := ToUInt256(amountAvailableInt256)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var getInputTokenAmount, getOutputTokenAmount func(target, current, liquidity *uint256.Int) (*uint256.Int, error)

	if zeroToOne {
		getInputTokenAmount = getInputTokenDelta01
		getOutputTokenAmount = getOutputTokenDelta01
	} else {
		getInputTokenAmount = getInputTokenDelta10
		getOutputTokenAmount = getOutputTokenDelta10
	}

	var (
		resultPrice, input, output, feeAmount *uint256.Int
	)

	feeDenoMinusFee := new(uint256.Int).SubUint64(FEE_DENOMINATOR, uint64(fee))
	if amountAvailable.Sign() >= 0 {
		amountAvailableAfterFee, overflow := new(uint256.Int).MulDivOverflow(amountAvailable, feeDenoMinusFee,
			FEE_DENOMINATOR)
		if overflow {
			return nil, nil, nil, nil, ErrOverflow
		}

		input, err = getInputTokenAmount(targetPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if amountAvailableAfterFee.Cmp(input) >= 0 {
			resultPrice = targetPrice
			feeAmount, err = v3Utils.MulDivRoundingUp(input, uint256.NewInt(uint64(fee)), feeDenoMinusFee)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		} else {
			resultPrice, err = getNewPriceAfterInput(currentPrice, liquidity, amountAvailableAfterFee, zeroToOne)
			if err != nil {
				return nil, nil, nil, nil, err
			}

			if targetPrice.Cmp(resultPrice) == 0 {
				return nil, nil, nil, nil, fmt.Errorf("target price should not equal result price")
			}

			input, err = getInputTokenAmount(resultPrice, currentPrice, liquidity)
			if err != nil {
				return nil, nil, nil, nil, err
			}

			feeAmount = new(uint256.Int).Sub(amountAvailable, input)
		}

		output, err = getOutputTokenAmount(resultPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

	} else {
		output, err = getOutputTokenAmount(targetPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		amountAvailable.Neg(amountAvailable)
		if amountAvailable.Sign() < 0 {
			return nil, nil, nil, nil, ErrInvalidAmountRequired
		}

		if amountAvailable.Cmp(output) >= 0 {
			resultPrice = targetPrice
		} else {
			resultPrice, err = getNewPriceAfterOutput(currentPrice, liquidity, amountAvailable, zeroToOne)
			if err != nil {
				return nil, nil, nil, nil, err
			}

			if targetPrice.Cmp(resultPrice) != 0 {
				output, err = getOutputTokenAmount(resultPrice, currentPrice, liquidity)
				if err != nil {
					return nil, nil, nil, nil, err
				}
			}

			if output.Cmp(amountAvailable) > 0 {
				output.Set(amountAvailable)
			}
		}

		input, err = getInputTokenAmount(resultPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		feeAmount, err = v3Utils.MulDivRoundingUp(input, uint256.NewInt(uint64(fee)),
			feeDenoMinusFee)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return resultPrice, input, output, feeAmount, nil
}
