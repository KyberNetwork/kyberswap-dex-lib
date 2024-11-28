package integral

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/daoleno/uniswapv3-sdk/utils"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	globalState          GlobalState
	liquidity            *big.Int
	nextTickGlobal       int32
	prevTickGlobal       int32
	totalFeeGrowth0Token *big.Int
	totalFeeGrowth1Token *big.Int

	ticks       *TickManager
	gas         int64
	tickMin     int
	tickMax     int
	tickSpacing int

	plugin    VotatilityOracle
	feeConfig FeeConfiguration

	fee uint16
}

type VotatilityOracle struct {
	timepoints             *TimepointStorage
	timepointIndex         uint16
	lastTimepointTimestamp uint32
	isInitialized          bool
}

func NewPoolSimulator(entityPool entity.Pool, defaultGas int64) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
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

	if !extra.GlobalState.Unlocked {
		return nil, ErrPoolLocked
	}

	ticks := NewTickManager()

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
		Pool:        pool.Pool{Info: info},
		globalState: extra.GlobalState,
		liquidity:   extra.Liquidity,
		ticks:       ticks,
		gas:         defaultGas,
		tickMin:     tickMin,
		tickMax:     tickMax,
		tickSpacing: int(extra.TickSpacing),
	}, nil
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool) *big.Int {
	var tickLimit int
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	sqrtPriceX96Limit, err := v3Utils.GetSqrtRatioAtTick(tickLimit)

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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(tokenOut, p.Info.Tokens[0]) {
			zeroForOne = false
		} else {
			zeroForOne = true
		}

		priceLimit := p.getSqrtPriceLimit(zeroForOne)
		err, amount0, amount1, stateUpdate := p._calculateSwapAndLock(zeroForOne, tokenAmountIn.Amount, priceLimit)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}

		var amountOut *big.Int
		if zeroForOne {
			amountOut = new(big.Int).Neg(amount1)
		} else {
			amountOut = new(big.Int).Neg(amount0)
		}

		if amountOut.Cmp(integer.Zero()) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenAmountIn.Token,
					Amount: nil,
				},
				Gas:      p.gas,
				SwapInfo: *stateUpdate,
			}, nil
		}

		return &pool.CalcAmountOutResult{}, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
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
	lastIndex := p.plugin.timepointIndex
	lastTimepointTimestamp := p.plugin.lastTimepointTimestamp

	if !p.plugin.isInitialized {
		return errors.New("Not initialized")
	}

	currentTimestamp := time.Now().Unix()
	if lastTimepointTimestamp == uint32(currentTimestamp) {
		return nil
	}

	tick := p.globalState.Tick
	newLastIndex, _, err := p.plugin.timepoints.write(lastIndex, uint32(currentTimestamp), tick)
	if err != nil {
		return err
	}

	p.plugin.timepointIndex = newLastIndex
	p.plugin.lastTimepointTimestamp = uint32(currentTimestamp)

	return nil
}

func (p *PoolSimulator) beforeSwap(zeroToOne bool, amountRequired, a *big.Int) (uint32, uint32, error) {
	if p.globalState.PluginConfig&BEFORE_SWAP_FLAG != 0 {
		if err := p.writeTimepoint(); err != nil {
			return 0, 0, err
		}

		votatilityAverage, err := p.getAverageVotatilityLast()
		if err != nil {
			return 0, 0, err
		}

		var newFee uint16
		if p.feeConfig.Alpha1|p.feeConfig.Alpha2 == 0 {
			newFee = p.feeConfig.BaseFee
		} else {
			newFee = getFee(votatilityAverage, p.feeConfig)
		}

		if newFee != p.fee {
			p.fee = newFee
		}
	}

	return 0, 0, nil
}

func (p *PoolSimulator) getAverageVotatilityLast() (*big.Int, error) {
	currentTimestamp := uint32(time.Now().Unix())

	tick := p.globalState.Tick
	lastTimepointIndex := p.plugin.timepointIndex
	oldestIndex := p.plugin.timepoints.getOldestIndex(lastTimepointIndex)

	votatilityAverage, err := p.plugin.timepoints.getAverageVolatility(currentTimestamp, tick, lastTimepointIndex, oldestIndex)
	if err != nil {
		return nil, err
	}

	return votatilityAverage, nil
}

func (p *PoolSimulator) swap(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var zeroForOne bool

	overrideFee, pluginFee, err := p.beforeSwap(zeroForOne, nil, nil)
	if err != nil {
		return nil, err
	}

	var eventParams SwapEventParams

	priceLimit := p.getSqrtPriceLimit(zeroForOne)

	amount0, amount1, currentPrice, currentTick, currentLiquidity, fees, err := p.calculateSwap(
		overrideFee, pluginFee, zeroForOne, tokenAmountIn.Amount, priceLimit)

	if err != nil {
		return nil, err
	}

	eventParams.currentPrice = currentPrice
	eventParams.currentTick = currentTick
	eventParams.currentLiquidity = currentLiquidity

	reserves := p.GetReserves()
	balance0Before, balance1Before := reserves[tokenInIndex], reserves[tokenOutIndex]
	if !zeroForOne {
		balance0Before, balance1Before = reserves[tokenOutIndex], reserves[tokenInIndex]
	}

	if zeroForOne {
		if amount1.Sign() < 0 {
			amount1.Neg(amount1)
		}

	} else {
		if amount0.Sign() < 0 {
			amount0.Neg(amount0)
		}
	}

	return &pool.CalcAmountOutResult{}, nil

}

func (p *PoolSimulator) calculateSwap(overrideFee, pluginFee uint32, zeroToOne bool, amountRequired, limitSqrtPrice *big.Int) (
	*big.Int, *big.Int, *big.Int, int32, *big.Int, FeesAmount, error) {
	if amountRequired.Sign() == 0 {
		// zeroAmountRequired();
		return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("----")
	}

	if amountRequired == MIN_INT256 {
		// invalidAmountRequired();
		return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("-------")
	}

	var cache SwapCalculationCache
	var fees FeesAmount

	cache.amountRequiredInitial = amountRequired
	cache.exactInput = amountRequired.Sign() > 0
	cache.pluginFee = pluginFee

	currentLiquidity := p.liquidity
	cache.prevInitializedTick = p.prevTickGlobal
	cache.nextInitializedTick = p.nextTickGlobal

	currentPrice := p.globalState.Price
	currentTick := p.globalState.Tick
	cache.fee = uint32(p.globalState.LastFee)
	cache.communityFee = big.NewInt(int64(p.globalState.CommunityFee))

	if currentPrice.Sign() == 0 {
		// revert notInitialized();
		return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("-------")
	}

	if overrideFee != 0 {
		cache.fee = overrideFee + pluginFee
	} else {
		if pluginFee != 0 {
			cache.fee += pluginFee
		}

		if cache.fee > 1e6 {
			// revert incorrectPluginFee();
			return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("-------")
		}
	}

	if zeroToOne {
		if limitSqrtPrice.Cmp(currentPrice) >= 0 || limitSqrtPrice.Cmp(MIN_SQRT_RATIO) <= 0 {
			// revert invalidLimitSqrtPrice();
			return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("-------")
		}
		cache.totalFeeGrowthInput = p.totalFeeGrowth0Token
	} else {
		if limitSqrtPrice.Cmp(currentPrice) <= 0 || limitSqrtPrice.Cmp(MAX_SQRT_RATIO) >= 0 {
			// invalidLimitSqrtPrice();
			return nil, nil, nil, 0, nil, FeesAmount{}, errors.New("-------")
		}
		cache.totalFeeGrowthInput = p.totalFeeGrowth1Token
	}

	var step PriceMovementCache
	for amountRequired.Sign() != 0 && currentPrice.Cmp(limitSqrtPrice) != 0 {
		var nextTick = cache.nextInitializedTick
		if zeroToOne {
			nextTick = cache.prevInitializedTick
		}

		step.stepSqrtPrice = currentPrice
		nextTickPrice, err := utils.GetSqrtRatioAtTick(int(nextTick))
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}
		step.nextTickPrice = nextTickPrice

		var targetPrice = step.nextTickPrice
		if zeroToOne == (step.nextTickPrice.Cmp(limitSqrtPrice) < 0) {
			targetPrice = limitSqrtPrice
		}

		currentPrice, input, output, feeAmount, err := movePriceTowardsTarget(zeroToOne, currentPrice, targetPrice, currentLiquidity, amountRequired, cache.fee)
		if err != nil {
			return nil, nil, nil, 0, nil, FeesAmount{}, err
		}

		step.input = input
		step.output = output
		step.feeAmount = feeAmount

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

		if currentLiquidity.Sign() > 0 {
			feeGrowthInput, err := mulDiv(step.feeAmount, Q128, currentLiquidity)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			cache.totalFeeGrowthInput.Add(cache.totalFeeGrowthInput, feeGrowthInput)
		}

		if currentPrice.Cmp(step.nextTickPrice) == 0 {
			if !cache.crossedAnyTick {
				cache.crossedAnyTick = true
				cache.totalFeeGrowthOutput = p.totalFeeGrowth0Token
				if zeroToOne {
					cache.totalFeeGrowthOutput = p.totalFeeGrowth1Token
				}

				var (
					liquidityDelta      *big.Int
					prevInitializedTick int32
					nextInitializedTick int32
				)

				if zeroToOne {
					liquidityDelta, prevInitializedTick, _ = p.ticks.cross(nextTick, cache.totalFeeGrowthInput, cache.totalFeeGrowthOutput)

					cache.prevInitializedTick = prevInitializedTick
					liquidityDelta.Neg(liquidityDelta)

					currentTick = nextTick - 1
					cache.nextInitializedTick = nextTick
				} else {
					liquidityDelta, _, nextInitializedTick = p.ticks.cross(nextTick, cache.totalFeeGrowthOutput, cache.totalFeeGrowthInput)

					cache.nextInitializedTick = nextInitializedTick
					currentTick = nextTick
					cache.prevInitializedTick = nextTick
				}

				currentLiquidity, err = addDelta(currentLiquidity, liquidityDelta)
				if err != nil {
					return nil, nil, nil, 0, nil, FeesAmount{}, err
				}
			}
		} else if currentPrice.Cmp(step.stepSqrtPrice) != 0 {
			currentTickInt, err := utils.GetTickAtSqrtRatio(currentPrice)
			if err != nil {
				return nil, nil, nil, 0, nil, FeesAmount{}, err
			}

			currentTick = int32(currentTickInt)

			break
		}
	}

	amountSpent := new(big.Int).Sub(cache.amountRequiredInitial, amountRequired)

	amount0, amount1 := cache.amountCalculated, amountSpent
	if zeroToOne == cache.exactInput {
		amount0, amount1 = amountSpent, cache.amountCalculated
	}

	p.globalState.Price = currentPrice
	p.globalState.Tick = currentTick

	if cache.crossedAnyTick {
		p.liquidity, p.prevTickGlobal, p.nextTickGlobal = currentLiquidity, cache.prevInitializedTick, cache.nextInitializedTick
	}

	if zeroToOne {
		p.totalFeeGrowth0Token = cache.totalFeeGrowthInput
	} else {
		p.totalFeeGrowth1Token = cache.totalFeeGrowthOutput
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
			feeAmount, err = mulDivRoundingUp(input, big.NewInt(int64(fee)), new(big.Int).Sub(FEE_DENOMINATOR, big.NewInt(int64(fee))))
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

		feeAmount, err = mulDivRoundingUp(input, big.NewInt(int64(fee)), new(big.Int).Sub(FEE_DENOMINATOR, big.NewInt(int64(fee))))
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return resultPrice, input, output, feeAmount, nil
}
