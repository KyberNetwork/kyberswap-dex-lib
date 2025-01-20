package integral

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
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
		liquidity:          extra.Liquidity,
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
	tokenAmtIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmtIn.Token
	amtRequired, overflow := uint256.FromBig(tokenAmtIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountRequired
	}

	amtSpent, amtCalculated, fees, gas, stateUpdate, err := p.swap(tokenIn, tokenOut, amtRequired)
	if err != nil {
		return nil, err
	} else if amtCalculated.IsZero() {
		return nil, ErrZeroAmountCalculated
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amtCalculated.Neg(amtCalculated).ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amtRequired.Sub(amtRequired, amtSpent).ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: new(uint256.Int).Add(fees.communityFeeAmount, fees.pluginFeeAmount).ToBig(),
		},
		Gas:      gas,
		SwapInfo: stateUpdate,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmtOut := param.TokenIn, param.TokenAmountOut
	tokenOut := tokenAmtOut.Token
	amtRequired, overflow := uint256.FromBig(tokenAmtOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountRequired
	}

	amtSpent, amtCalculated, fees, gas, stateUpdate, err := p.swap(tokenIn, tokenOut, amtRequired.Neg(amtRequired))
	if err != nil {
		return nil, err
	} else if amtCalculated.IsZero() {
		return nil, ErrZeroAmountCalculated
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amtCalculated.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amtRequired.Neg(amtRequired.Sub(amtRequired, amtSpent)).ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: new(uint256.Int).Add(fees.communityFeeAmount, fees.pluginFeeAmount).ToBig(),
		},
		Gas:      gas,
		SwapInfo: stateUpdate,
	}, nil
}

func (p *PoolSimulator) swap(tokenIn, tokenOut string, amtRequired *uint256.Int) (amtSpent, amtCalculated *uint256.Int,
	fees FeesAmount, gas int64, stateUpdate StateUpdate, err error) {
	if !p.globalState.Unlocked {
		err = ErrPoolLocked
		return
	}

	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		err = ErrInvalidToken
		return
	}

	zeroForOne := tokenInIndex == 0
	overrideFee, pluginFee, err := lo.Ternary(p.useBasePluginV2, p.beforeSwapV2, p.beforeSwapV1)(zeroForOne)
	if err != nil {
		return
	}

	priceLimit, err := p.getSqrtPriceLimit(zeroForOne)
	if err != nil {
		return
	}

	amtSpent, amtCalculated, currentPrice, currentTick, currentLiquidity, fees, err := p.calculateSwap(
		overrideFee, pluginFee, zeroForOne, amtRequired, priceLimit)
	if err != nil {
		return
	}

	return amtSpent, amtCalculated, fees, p.gas, StateUpdate{
		Liquidity: currentLiquidity,
		Price:     currentPrice,
		Tick:      currentTick,
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
	p.liquidity = si.Liquidity
	p.globalState.Price = si.Price
	p.globalState.Tick = si.Tick
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

	if tickLimit == v3Utils.MinTick || tickLimit == v3Utils.MaxTick {
		lo.Ternary(zeroForOne, sqrtPriceX96Limit.AddUint64, sqrtPriceX96Limit.SubUint64)(&sqrtPriceX96Limit, 1)
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

func (p *PoolSimulator) beforeSwapV1(_ bool) (uint32, uint32, error) {
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

	if adjustedFee.BitLen() > 15 || adjustedFee.IsZero() {
		adjustedFee.SetUint64(math.MaxUint16)
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

func (p *PoolSimulator) calculateSwap(overrideFee, pluginFee uint32, zeroToOne bool, amountRequired *uint256.Int,
	limitSqrtPrice *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int, int32, *uint256.Int, FeesAmount, error) {
	if amountRequired.IsZero() {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrZeroAmountRequired
	}

	var cache SwapCalculationCache
	var fees = FeesAmount{
		communityFeeAmount: new(uint256.Int),
		pluginFeeAmount:    new(uint256.Int),
	}

	cache.amountRequiredInitial = amountRequired
	cache.exactInput = amountRequired.Sign() > 0
	cache.amountCalculated = new(uint256.Int)

	currentLiquidity := p.liquidity
	currentPrice := p.globalState.Price
	if currentPrice.IsZero() {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrNotInitialized
	}
	currentTick := p.globalState.Tick

	if pluginFee > 0 {
		cache.pluginFee = uint256.NewInt(uint64(pluginFee))
	}
	if overrideFee != 0 {
		cache.fee = uint64(overrideFee + pluginFee)
	} else if fee := uint32(p.globalState.LastFee) + pluginFee; fee != 0 {
		if fee >= 1e6 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrIncorrectPluginFee
		}
		cache.fee = uint64(fee)
	}
	feeU := uint256.NewInt(cache.fee)
	cache.communityFee = uint256.NewInt(uint64(p.globalState.CommunityFee))

	if zeroToOne && limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(MIN_SQRT_RATIO) <= 0 ||
		!zeroToOne && limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(MAX_SQRT_RATIO) >= 0 {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidLimitSqrtPrice
	}

	step := PriceMovementCache{nextTickPrice: new(uint256.Int)}
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
		if err = v3Utils.GetSqrtRatioAtTickV2(int(step.nextTick), step.nextTickPrice); err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		var targetPrice = step.nextTickPrice
		if zeroToOne == (step.nextTickPrice.Cmp(limitSqrtPrice) < 0) {
			targetPrice = limitSqrtPrice
		}

		currentPrice, step.input, step.output, step.feeAmount, err = movePriceTowardsTarget(zeroToOne, currentPrice,
			targetPrice, currentLiquidity, amountRequired, cache.fee)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		if cache.exactInput {
			amountRequired.Sub(amountRequired, step.input).Sub(amountRequired, step.feeAmount)
			cache.amountCalculated.Sub(cache.amountCalculated, step.output)
		} else {
			amountRequired.Add(amountRequired, step.output)
			cache.amountCalculated.Add(cache.amountCalculated, step.input).Add(cache.amountCalculated, step.feeAmount)
		}

		if cache.pluginFee != nil && cache.fee > 0 {
			delta, err := v3Utils.MulDiv(step.feeAmount, cache.pluginFee, feeU)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.pluginFeeAmount.Add(fees.pluginFeeAmount, delta)
		}

		if cache.communityFee.Sign() > 0 {
			delta, err := v3Utils.MulDiv(step.feeAmount, cache.communityFee, COMMUNITY_FEE_DENOMINATOR)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.communityFeeAmount.Add(fees.communityFeeAmount, delta)
		}

		if currentPrice.Cmp(step.nextTickPrice) == 0 {
			tickData, err := p.ticks.GetTick(int(step.nextTick))
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			var liquidityDelta *int256.Int
			if zeroToOne {
				currentTick = step.nextTick - 1
				initializedTick = step.nextTick - 1
				liquidityDelta = new(int256.Int).Neg(tickData.LiquidityNet)
			} else {
				currentTick = step.nextTick
				initializedTick = step.nextTick
				liquidityDelta = tickData.LiquidityNet
			}

			currentLiquidity, err = addDelta(currentLiquidity, liquidityDelta)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

		} else if currentPrice.Cmp(step.stepSqrtPrice) != 0 {
			currentTickInt, err := v3Utils.GetTickAtSqrtRatioV2(currentPrice)
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

	amountSpent := new(uint256.Int).Sub(cache.amountRequiredInitial, amountRequired)
	return amountSpent, cache.amountCalculated, currentPrice, currentTick, currentLiquidity, fees, nil
}

func movePriceTowardsTarget(zeroToOne bool, currentPrice, targetPrice, liquidity, amountAvailable *uint256.Int,
	fee uint64) (resultPrice, input, output, feeAmount *uint256.Int, err error) {
	getInputTokenAmount, getOutputTokenAmount := getInputTokenDelta10, getOutputTokenDelta10
	if zeroToOne {
		getInputTokenAmount, getOutputTokenAmount = getInputTokenDelta01, getOutputTokenDelta01
	}

	feeDenoMinusFee := new(uint256.Int).SubUint64(FEE_DENOMINATOR, fee)
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
			feeAmount, err = v3Utils.MulDivRoundingUp(input, uint256.NewInt(fee), feeDenoMinusFee)
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

		amountAvailable = new(uint256.Int).Neg(amountAvailable)
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

		feeAmount, err = v3Utils.MulDivRoundingUp(input, uint256.NewInt(fee),
			feeDenoMinusFee)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return resultPrice, input, output, feeAmount, nil
}
