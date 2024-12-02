package integral

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/daoleno/uniswapv3-sdk/utils"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	globalState GlobalState
	liquidity   *big.Int

	ticks       *v3Entities.TickListDataProvider
	gas         int64
	tickMin     int32
	tickMax     int32
	tickSpacing int

	volatilityOracle *VotatilityOraclePlugin
	dynamicFee       *DynamicFeePlugin
	slidingFee       *SlidingFeePlugin

	useBasePluginV2 bool
}

type VotatilityOraclePlugin struct {
	Timepoints             TimepointStorage
	TimepointIndex         uint16
	LastTimepointTimestamp uint32
	IsInitialized          bool
}

type DynamicFeePlugin struct {
	FeeConfig FeeConfiguration
}

type SlidingFeePlugin struct {
	FeeFactors FeeFactors
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

	tickMin := extra.Ticks[0].Index
	tickMax := extra.Ticks[len(extra.Ticks)-1].Index

	var info = pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:             pool.Pool{Info: info},
		globalState:      extra.GlobalState,
		liquidity:        extra.Liquidity,
		ticks:            ticks,
		gas:              defaultGas,
		tickMin:          int32(tickMin),
		tickMax:          int32(tickMax),
		tickSpacing:      int(extra.TickSpacing),
		volatilityOracle: &extra.VotatilityOracle,
		dynamicFee:       &extra.DynamicFee,
		slidingFee:       &extra.SlidingFee,
		useBasePluginV2:  staticExtra.UseBasePluginV2,
	}, nil
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool) *big.Int {
	var tickLimit int32
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	sqrtPriceX96Limit, err := v3Utils.GetSqrtRatioAtTick(int(tickLimit))

	if zeroForOne {
		sqrtPriceX96Limit = new(big.Int).Add(sqrtPriceX96Limit, integer.One()) // = (sqrtPrice at minTick) + 1
	} else {
		sqrtPriceX96Limit = new(big.Int).Sub(sqrtPriceX96Limit, integer.One()) // = (sqrtPrice at maxTick) - 1
	}

	if err != nil {
		return nil
	}

	return sqrtPriceX96Limit
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(StateUpdate)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Algebra %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}
	p.liquidity = new(big.Int).Set(si.Liquidity)
	p.globalState = si.GlobalState
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	zeroForOne := strings.EqualFold(tokenIn, p.Info.Tokens[0])
	return PoolMeta{
		BlockNumber: p.Pool.Info.BlockNumber,
		PriceLimit:  p.getSqrtPriceLimit(zeroForOne),
	}
}

func (p *PoolSimulator) writeTimepoint() error {
	lastIndex := p.volatilityOracle.TimepointIndex
	lastTimepointTimestamp := p.volatilityOracle.LastTimepointTimestamp

	if !p.volatilityOracle.IsInitialized {
		return ErrNotInitialized
	}

	currentTimestamp := time.Now().Unix()
	if lastTimepointTimestamp == uint32(currentTimestamp) {
		return nil
	}

	tick := p.globalState.Tick
	newLastIndex, _, err := p.volatilityOracle.Timepoints.write(lastIndex, uint32(currentTimestamp), tick)
	if err != nil {
		return err
	}

	p.volatilityOracle.TimepointIndex = newLastIndex
	p.volatilityOracle.LastTimepointTimestamp = uint32(currentTimestamp)

	return nil
}

func (p *PoolSimulator) beforeSwapV1() (uint32, uint32, error) {
	if p.globalState.PluginConfig&BEFORE_SWAP_FLAG != 0 {
		if err := p.writeTimepoint(); err != nil {
			return 0, 0, err
		}

		votatilityAverage, err := p.getAverageVotatilityLast()
		if err != nil {
			return 0, 0, err
		}

		var newFee uint16
		if p.dynamicFee.FeeConfig.Alpha1|p.dynamicFee.FeeConfig.Alpha2 == 0 {
			newFee = p.dynamicFee.FeeConfig.BaseFee
		} else {
			newFee = getFee(votatilityAverage, p.dynamicFee.FeeConfig)
		}

		if newFee != p.globalState.LastFee {
			p.globalState.LastFee = newFee
		}
	}

	return 0, 0, nil
}

func (p *PoolSimulator) beforeSwapV2(zeroToOne bool) (uint32, uint32, error) {
	currentTick := p.globalState.Tick
	lastTick := p.getLastTick()

	newFee, err := p.getFeeAndUpdateFactors(zeroToOne, currentTick, lastTick)
	if err != nil {
		return 0, 0, err
	}

	if err := p.writeTimepoint(); err != nil {
		return 0, 0, err
	}

	return uint32(newFee), 0, nil
}

func (p *PoolSimulator) getFeeAndUpdateFactors(zeroToOne bool, currentTick, lastTick int32) (uint16, error) {
	var currentFeeFactors FeeFactors

	if currentTick != lastTick {
		currentFeeFactors, err := calculateFeeFactors(currentTick, lastTick, s_priceChangeFactor)
		if err != nil {
			return 0, err
		}

		p.slidingFee.FeeFactors = currentFeeFactors
	} else {
		currentFeeFactors = p.slidingFee.FeeFactors
	}

	var adjustedFee *big.Int
	baseFeeBig := big.NewInt(s_baseFee)

	if zeroToOne {
		adjustedFee = new(big.Int).Rsh(
			new(big.Int).Mul(baseFeeBig, currentFeeFactors.zeroToOneFeeFactor),
			FEE_FACTOR_SHIFT,
		)
	} else {
		adjustedFee = new(big.Int).Rsh(
			new(big.Int).Mul(baseFeeBig, currentFeeFactors.oneToZeroFeeFactor),
			FEE_FACTOR_SHIFT,
		)
	}

	if adjustedFee.Cmp(MAX_UINT16) > 0 {
		adjustedFee.Set(MAX_UINT16)
	} else if adjustedFee.Sign() == 0 {
		adjustedFee.Set(MAX_UINT16)
	}

	return uint16(adjustedFee.Int64()), nil
}

func (p *PoolSimulator) getLastTick() int32 {
	lastTimepointIndex := p.volatilityOracle.TimepointIndex
	lastTimepoint := p.volatilityOracle.Timepoints.Get(lastTimepointIndex)

	return lastTimepoint.Tick
}

func (p *PoolSimulator) getAverageVotatilityLast() (*big.Int, error) {
	currentTimestamp := uint32(time.Now().Unix())

	tick := p.globalState.Tick
	lastTimepointIndex := p.volatilityOracle.TimepointIndex
	oldestIndex := p.volatilityOracle.Timepoints.getOldestIndex(lastTimepointIndex)

	votatilityAverage, err := p.volatilityOracle.Timepoints.getAverageVolatility(currentTimestamp, tick, lastTimepointIndex, oldestIndex)
	if err != nil {
		return nil, err
	}

	return votatilityAverage, nil
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

	priceLimit := p.getSqrtPriceLimit(zeroForOne)

	amount0, amount1, currentPrice, currentTick, currentLiquidity, fees, err := p.calculateSwap(
		overrideFee, pluginFee, zeroForOne, tokenAmountIn.Amount, priceLimit)

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

	var amountOut *big.Int
	if zeroForOne {
		amountOut = amount1
		if amount1.Sign() < 0 {
			amountOut = amount1.Neg(amount1)
		}

	} else {
		amountOut = amount0
		if amount0.Sign() < 0 {
			amountOut = amount0.Neg(amount0)
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenAmountIn.Token,
			Amount: new(big.Int).Add(fees.communityFeeAmount, fees.pluginFeeAmount),
		},
		SwapInfo: StateUpdate{
			GlobalState: newState,
			Liquidity:   currentLiquidity,
		},
	}, nil

}

func (p *PoolSimulator) calculateSwap(overrideFee, pluginFee uint32, zeroToOne bool, amountRequired, limitSqrtPrice *big.Int) (
	*big.Int, *big.Int, *big.Int, int32, *big.Int, FeesAmount, error) {
	if amountRequired.Sign() == 0 {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrZeroAmountRequired
	}

	if amountRequired.Cmp(MIN_INT256) >= 0 {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidAmountRequired
	}

	var cache SwapCalculationCache
	var fees FeesAmount

	cache.amountRequiredInitial = amountRequired
	cache.exactInput = amountRequired.Sign() > 0
	cache.pluginFee = pluginFee

	currentLiquidity := p.liquidity

	currentPrice := p.globalState.Price
	currentTick := p.globalState.Tick
	cache.fee = uint32(p.globalState.LastFee)
	cache.communityFee = big.NewInt(int64(p.globalState.CommunityFee))

	if currentPrice.Sign() == 0 {
		return nil, nil, nil, 0, nil, FeesAmount{}, ErrNotInitialized
	}

	if overrideFee != 0 {
		cache.fee = overrideFee + pluginFee
	} else {
		if pluginFee != 0 {
			cache.fee += pluginFee
		}

		if cache.fee > 1e6 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrIncorrectPluginFee
		}
	}

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(MIN_SQRT_RATIO) <= 0 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidLimitSqrtPrice
		}

	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(MAX_SQRT_RATIO) >= 0 {
			return nil, nil, nil, 0, nil, FeesAmount{}, ErrInvalidLimitSqrtPrice
		}

	}

	var step PriceMovementCache
	// swap until there is remaining input or output tokens or we reach the price limit
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		var (
			nextTick int
			err      error
		)

		nextTick, step.initialized, err = p.ticks.NextInitializedTickWithinOneWord(int(currentTick), zeroToOne, p.tickSpacing)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}
		step.nextTick = int32(nextTick)

		step.stepSqrtPrice = currentPrice
		step.nextTickPrice, err = utils.GetSqrtRatioAtTick(int(step.nextTick))
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		var targetPrice = step.nextTickPrice
		if zeroToOne == (step.nextTickPrice.Cmp(limitSqrtPrice) < 0) {
			targetPrice = limitSqrtPrice
		}

		currentPrice, step.input, step.output, step.feeAmount, err = movePriceTowardsTarget(zeroToOne, currentPrice, targetPrice, currentLiquidity, amountRequired, cache.fee)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		if cache.exactInput {
			amountRequired.Sub(amountRequired, new(big.Int).Add(step.input, step.feeAmount))
			cache.amountCalculated = new(big.Int).Sub(cache.amountCalculated, step.output)
		} else {
			amountRequired.Add(amountRequired, step.output)
			cache.amountCalculated = new(big.Int).Add(cache.amountCalculated, new(big.Int).Add(step.input, step.feeAmount))
		}

		if cache.pluginFee > 0 && cache.fee > 0 {
			delta, err := mulDiv(step.feeAmount, big.NewInt(int64(cache.pluginFee)), big.NewInt(int64(cache.fee)))
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.pluginFeeAmount.Add(fees.pluginFeeAmount, delta)
		}

		if cache.communityFee.Sign() > 0 {
			delta := new(big.Int).Div(
				new(big.Int).Mul(step.feeAmount, cache.communityFee),
				COMMUNITY_FEE_DENOMINATOR,
			)

			step.feeAmount.Sub(step.feeAmount, delta)
			fees.communityFeeAmount.Add(fees.communityFeeAmount, delta)
		}

		if currentPrice.Cmp(step.nextTickPrice) == 0 {
			if step.initialized {
				tickData, err := p.ticks.GetTick(int(step.nextTick))
				if err != nil {
					return nil, nil, nil, 0, nil, FeesAmount{}, err
				}

				var liquidityDelta *big.Int
				if zeroToOne {
					liquidityDelta = new(big.Int).Neg(tickData.LiquidityNet)
				} else {
					liquidityDelta = tickData.LiquidityNet
				}

				currentLiquidity = utils.AddDelta(currentLiquidity, liquidityDelta)
			}

			if zeroToOne {
				currentTick = step.nextTick - 1
			} else {
				currentTick = step.nextTick
			}

		} else if currentPrice.Cmp(step.stepSqrtPrice) != 0 {
			currentTickInt, err := utils.GetTickAtSqrtRatio(currentPrice)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			currentTick = int32(currentTickInt)

			break
		}

		if amountRequired.Sign() == 0 || currentPrice.Cmp(limitSqrtPrice) == 0 {
			break
		}
	}

	amountSpent := new(big.Int).Sub(cache.amountRequiredInitial, amountRequired)

	amount0, amount1 := cache.amountCalculated, amountSpent
	if zeroToOne == cache.exactInput {
		amount0, amount1 = amountSpent, cache.amountCalculated
	}

	return amount0, amount1, currentPrice, currentTick, currentLiquidity, fees, nil
}

func movePriceTowardsTarget(
	zeroToOne bool,
	currentPrice, targetPrice, liquidity, amountAvailable *big.Int,
	fee uint32,
) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	var getInputTokenAmount, getOutputTokenAmount func(target, current *big.Int, liquidity *big.Int) (*big.Int, error)
	if zeroToOne {
		getInputTokenAmount = getInputTokenDelta01
		getOutputTokenAmount = getOutputTokenDelta01
	} else {
		getInputTokenAmount = getInputTokenDelta10
		getOutputTokenAmount = getOutputTokenDelta10
	}

	var (
		resultPrice, input, output, feeAmount *big.Int
		err                                   error
	)

	if amountAvailable.Sign() >= 0 {
		// amountAvailableAfterFee = amountAvailable * (FEE_DENOMINATOR - fee) / FEE_DENOMINATOR
		amountAvailableAfterFee := new(big.Int).Mul(amountAvailable, new(big.Int).Sub(FEE_DENOMINATOR, big.NewInt(int64(fee))))
		amountAvailableAfterFee.Div(amountAvailableAfterFee, FEE_DENOMINATOR)

		input, err = getInputTokenAmount(targetPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if amountAvailableAfterFee.Cmp(input) >= 0 {
			resultPrice = new(big.Int).Set(targetPrice)
			feeAmount = utils.MulDivRoundingUp(input, big.NewInt(int64(fee)), new(big.Int).Sub(FEE_DENOMINATOR, big.NewInt(int64(fee))))
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

			feeAmount = new(big.Int).Sub(amountAvailable, input)
		}

		output, err = getOutputTokenAmount(resultPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

	} else {
		amountAvailable.Neg(amountAvailable)

		output, err = getOutputTokenAmount(targetPrice, currentPrice, liquidity)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if amountAvailable.Cmp(output) >= 0 {
			resultPrice = new(big.Int).Set(targetPrice)
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

		feeAmount = utils.MulDivRoundingUp(input, big.NewInt(int64(fee)), new(big.Int).Sub(FEE_DENOMINATOR, big.NewInt(int64(fee))))
	}

	return resultPrice, input, output, feeAmount, nil
}
