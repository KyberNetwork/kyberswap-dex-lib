package integral

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	globalState GlobalState
	liquidity   *uint256.Int

	ticks   *v3Entities.TickListDataProvider
	tickMin int32
	tickMax int32

	timepoints         *TimepointStorage
	volatilityOracle   *VolatilityOraclePlugin
	dynamicFee         *DynamicFeeConfig
	slidingFee         *SlidingFeeConfig
	writeTimePointOnce *sync.Once

	useBasePluginV2 bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
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

	ticks, err := v3Entities.NewTickListDataProvider(extra.Ticks, 1)
	if err != nil {
		return nil, err
	}

	tickMin := extra.Ticks[0].Index
	tickMax := extra.Ticks[len(extra.Ticks)-1].Index

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		globalState:        extra.GlobalState,
		liquidity:          extra.Liquidity,
		ticks:              ticks,
		tickMin:            int32(tickMin),
		tickMax:            int32(tickMax),
		timepoints:         NewTimepointStorage(extra.Timepoints),
		volatilityOracle:   extra.VolatilityOracle,
		dynamicFee:         extra.DynamicFee,
		slidingFee:         extra.SlidingFee,
		writeTimePointOnce: new(sync.Once),
		useBasePluginV2:    staticExtra.UseBasePluginV2,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmtIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmtIn.Token
	var amtRequired uint256.Int
	if amtRequired.SetFromBig(tokenAmtIn.Amount) {
		return nil, ErrInvalidAmountRequired
	}

	b := &calcAmountOutBacking{}
	if err := p.swap(tokenIn, tokenOut, &amtRequired, &b.o); err != nil {
		return nil, err
	} else if b.o.calculated.IsZero() {
		return nil, ErrZeroAmountCalculated
	}

	b.o.calculated.Neg(&b.o.calculated)
	b.out = pool.TokenAmount{Token: tokenOut, Amount: big256.ToBigBacked(&b.outBI, &b.outWords, &b.o.calculated)}
	b.remaining = pool.TokenAmount{Token: tokenIn,
		Amount: big256.ToBigBacked(&b.remainingBI, &b.remainingWords, &b.o.remaining)}
	b.fee = pool.TokenAmount{Token: tokenIn, Amount: big256.ToBigBacked(&b.feeBI, &b.feeWords, &b.o.fee)}
	b.res = pool.CalcAmountOutResult{
		TokenAmountOut:         &b.out,
		RemainingTokenAmountIn: &b.remaining,
		Fee:                    &b.fee,
		Gas:                    b.o.gas,
		SwapInfo:               b.o.stateUpdate(),
	}
	return &b.res, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmtOut := param.TokenIn, param.TokenAmountOut
	tokenOut := tokenAmtOut.Token
	var amtRequired uint256.Int
	if amtRequired.SetFromBig(tokenAmtOut.Amount) {
		return nil, ErrInvalidAmountRequired
	}

	b := &calcAmountInBacking{}
	if err := p.swap(tokenIn, tokenOut, amtRequired.Neg(&amtRequired), &b.o); err != nil {
		return nil, err
	} else if b.o.calculated.IsZero() {
		return nil, ErrZeroAmountCalculated
	}

	b.o.remaining.Neg(&b.o.remaining)
	b.in = pool.TokenAmount{Token: tokenIn, Amount: big256.ToBigBacked(&b.inBI, &b.inWords, &b.o.calculated)}
	b.remaining = pool.TokenAmount{Token: tokenOut,
		Amount: big256.ToBigBacked(&b.remainingBI, &b.remainingWords, &b.o.remaining)}
	b.fee = pool.TokenAmount{Token: tokenIn, Amount: big256.ToBigBacked(&b.feeBI, &b.feeWords, &b.o.fee)}
	b.res = pool.CalcAmountInResult{
		TokenAmountIn:           &b.in,
		RemainingTokenAmountOut: &b.remaining,
		Fee:                     &b.fee,
		Gas:                     b.o.gas,
		SwapInfo:                b.o.stateUpdate(),
	}
	return &b.res, nil
}

// swapOutcome is what a swap leaves: amounts, fees, gas and the pool state after it.
type swapOutcome struct {
	remaining  uint256.Int // amountRequired left unswapped, signed like the request
	calculated uint256.Int // signed: negative output for exact-in, positive input for exact-out
	fee        uint256.Int // community plus plugin fee
	price      uint256.Int
	liquidity  uint256.Int
	tick       int32
	gas        int64
}

func (o *swapOutcome) stateUpdate() StateUpdate {
	return StateUpdate{Liquidity: &o.liquidity, Price: &o.price, Tick: o.tick}
}

// calcAmountOutBacking holds everything a CalcAmountOut result points to, so a quote costs one
// allocation for it plus the SwapInfo interface box.
type calcAmountOutBacking struct {
	res                                pool.CalcAmountOutResult
	out, remaining, fee                pool.TokenAmount
	o                                  swapOutcome
	outBI, remainingBI, feeBI          big.Int
	outWords, remainingWords, feeWords [4]big.Word
}

// calcAmountInBacking is calcAmountOutBacking for CalcAmountIn.
type calcAmountInBacking struct {
	res                               pool.CalcAmountInResult
	in, remaining, fee                pool.TokenAmount
	o                                 swapOutcome
	inBI, remainingBI, feeBI          big.Int
	inWords, remainingWords, feeWords [4]big.Word
}

// swap quotes amtRequired (positive exact-in, negative exact-out) into o, consuming amtRequired.
func (p *PoolSimulator) swap(tokenIn, tokenOut string, amtRequired *uint256.Int, o *swapOutcome) error {
	if !p.globalState.Unlocked {
		return ErrPoolLocked
	}

	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return ErrInvalidToken
	}

	zeroForOne := tokenInIndex == 0
	var overrideFee, pluginFee uint32
	var err error
	if p.useBasePluginV2 && p.slidingFee.FeeType {
		overrideFee, pluginFee, err = p.beforeSwapV2(zeroForOne)
	} else {
		overrideFee, pluginFee, err = p.beforeSwapV1(zeroForOne)
	}
	if err != nil {
		return err
	}

	var priceLimit uint256.Int
	if err = p.sqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return err
	}
	return p.calculateSwap(overrideFee, pluginFee, zeroForOne, amtRequired, &priceLimit, o)
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

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	zeroForOne := strings.EqualFold(tokenIn, p.Info.Tokens[0])
	priceLimit, _ := p.getSqrtPriceLimit(zeroForOne)
	return PoolMeta{
		BlockNumber: p.Info.BlockNumber,
		PriceLimit:  priceLimit,
	}
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool) (*uint256.Int, error) {
	var sqrtPriceX96Limit uint256.Int
	if err := p.sqrtPriceLimit(zeroForOne, &sqrtPriceX96Limit); err != nil {
		return nil, err
	}
	return &sqrtPriceX96Limit, nil
}

func (p *PoolSimulator) sqrtPriceLimit(zeroForOne bool, sqrtPriceX96Limit *uint256.Int) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := uniswapv3.GetSqrtRatioAtTick(int(tickLimit), sqrtPriceX96Limit); err != nil {
		return err
	}
	if tickLimit == v3Utils.MinTick || tickLimit == v3Utils.MaxTick {
		if zeroForOne {
			sqrtPriceX96Limit.AddUint64(sqrtPriceX96Limit, 1)
		} else {
			sqrtPriceX96Limit.SubUint64(sqrtPriceX96Limit, 1)
		}
	}
	return nil
}

var blockTimestamp = func() uint32 { return uint32(time.Now().Unix()) }

// writeTimepoint locks and writes timepoint only once, triggering onWrite only if said write happened.
// By right we should re-update the timepoint every new second, but the difference should be small enough, and
// new pool should have already been created and used in replacement of this pool.
func (p *PoolSimulator) writeTimepoint(onWrite func() error) (err error) {
	p.writeTimePointOnce.Do(func() {
		if volatilityOracle := p.volatilityOracle; volatilityOracle != nil { // plugin uses volatility oracle
			if !volatilityOracle.IsInitialized {
				err = ErrNotInitialized
				return
			}
			volatilityOracle.LastTimepointTimestamp = blockTimestamp()
			volatilityOracle.TimepointIndex, _, err = p.timepoints.write(
				volatilityOracle.TimepointIndex, volatilityOracle.LastTimepointTimestamp, p.globalState.Tick)
		}
		if err != nil || onWrite == nil {
			return
		}
		err = onWrite()
	})
	return err
}

func (p *PoolSimulator) beforeSwapV1(zeroForOne bool) (uint32, uint32, error) {
	if p.globalState.PluginConfig&BEFORE_SWAP_FLAG == 0 || p.dynamicFee == nil {
		return 0, 0, nil
	}
	return 0, 0, p.writeTimepoint(func() error {
		var newFee uint16
		if p.dynamicFee.ZeroToOne != 0 || p.dynamicFee.OneToZero != 0 {
			// https://berascan.com/address/0x2393BcDBB298A4905f9885109B19834c50c8038F#code
			newFee = lo.Ternary(zeroForOne, p.dynamicFee.ZeroToOne, p.dynamicFee.OneToZero)
		} else if p.dynamicFee.Alpha1 == 0 && p.dynamicFee.Alpha2 == 0 {
			newFee = p.dynamicFee.BaseFee
		} else {
			volatilityLast, err := p.getAverageVolatilityLast()
			if err != nil {
				return err
			}
			newFee = getFee(volatilityLast, p.dynamicFee)
		}
		if newFee != 0 {
			p.globalState.LastFee = newFee
		}
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
		currentFeeFactors, err = p.calculateFeeFactors(currentTick, lastTick, p.slidingFee.PriceChangeFactor)
		if err != nil {
			return 0, err
		}

		p.slidingFee = currentFeeFactors
	} else {
		currentFeeFactors = p.slidingFee
	}

	adjustedFee := uint256.NewInt(uint64(p.slidingFee.BaseFee))
	adjustedFee = adjustedFee.Rsh(
		adjustedFee.Mul(adjustedFee,
			lo.Ternary(zeroToOne, currentFeeFactors.ZeroToOneFeeFactor, currentFeeFactors.OneToZeroFeeFactor),
		),
		FEE_FACTOR_SHIFT,
	)

	if adjustedFee.BitLen() > 16 {
		return math.MaxUint16, nil
	} else if adjustedFee.IsZero() {
		return 1, nil
	}
	return uint16(adjustedFee.Uint64()), nil
}

func (p *PoolSimulator) getLastTick() int32 {
	lastTimepointIndex := p.volatilityOracle.TimepointIndex
	lastTimepoint := p.timepoints.Get(lastTimepointIndex)
	return lastTimepoint.Tick
}

func (p *PoolSimulator) getAverageVolatilityLast() (*uint256.Int, error) {
	if p.volatilityOracle == nil {
		return nil, ErrNotInitialized
	}
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

func (p *PoolSimulator) calculateSwap(overrideFee, pluginFee uint32, zeroToOne bool, amountRequired,
	limitSqrtPrice *uint256.Int, o *swapOutcome) error {
	if amountRequired.IsZero() {
		return ErrZeroAmountRequired
	}
	exactInput := amountRequired.Sign() > 0

	currentLiquidity, currentPrice := &o.liquidity, &o.price
	currentLiquidity.Set(p.liquidity)
	currentPrice.Set(p.globalState.Price)
	if currentPrice.IsZero() {
		return ErrNotInitialized
	}
	currentTick := p.globalState.Tick

	fee := uint64(lo.Ternary(overrideFee != 0, overrideFee, uint32(p.globalState.LastFee)) + pluginFee)
	if fee >= 1e6 {
		return ErrIncorrectPluginFee
	}
	var pluginFeeU, feeU, communityFee uint256.Int
	pluginFeeU.SetUint64(uint64(pluginFee))
	feeU.SetUint64(fee)
	communityFee.SetUint64(uint64(p.globalState.CommunityFee))

	if zeroToOne && !limitSqrtPrice.Lt(currentPrice) || !limitSqrtPrice.Gt(MIN_SQRT_RATIO) ||
		!zeroToOne && !limitSqrtPrice.Gt(currentPrice) || !limitSqrtPrice.Lt(MAX_SQRT_RATIO) {
		return ErrInvalidLimitSqrtPrice
	}

	var step PriceMovementCache
	var communityFeeAmount, pluginFeeAmount, delta uint256.Int
	initializedTick := currentTick
	var ticksCrossed int64
	// swap until there is remaining input or output tokens, or we reach the price limit.
	// limit by maxSwapLoop to make sure we won't loop infinitely because of a bug somewhere
	for i := 0; i < maxSwapLoop; i++ {
		// Algebra crosses straight to the next tick in the list; the SDK's word-by-word search reached the
		// same tick, with the same errors past either end of the list. Every list tick counts as
		// initialized, as it did there.
		nextTick, _, err := p.ticks.NextInitializedTickIndex(int(initializedTick), zeroToOne)
		if err != nil {
			return err
		}
		step.nextTick = int32(nextTick)
		ticksCrossed++

		step.stepSqrtPrice.Set(currentPrice)
		if err = uniswapv3.GetSqrtRatioAtTick(nextTick, &step.nextTickPrice); err != nil {
			return err
		}

		targetPrice := &step.nextTickPrice
		if zeroToOne == step.nextTickPrice.Lt(limitSqrtPrice) {
			targetPrice = limitSqrtPrice
		}

		if err = movePriceTowardsTarget(zeroToOne, currentPrice, targetPrice, currentLiquidity, amountRequired, fee,
			&step.resultPrice, &step.input, &step.output, &step.feeAmount); err != nil {
			return err
		}
		currentPrice.Set(&step.resultPrice)

		if exactInput {
			amountRequired.Sub(amountRequired, &step.input).Sub(amountRequired, &step.feeAmount)
			o.calculated.Sub(&o.calculated, &step.output)
		} else {
			amountRequired.Add(amountRequired, &step.output)
			o.calculated.Add(&o.calculated, &step.input).Add(&o.calculated, &step.feeAmount)
		}

		if pluginFee > 0 && fee > 0 {
			if err = uniswapv3.MulDivV2(&step.feeAmount, &pluginFeeU, &feeU, &delta, nil); err != nil {
				return err
			}
			step.feeAmount.Sub(&step.feeAmount, &delta)
			pluginFeeAmount.Add(&pluginFeeAmount, &delta)
		}

		if communityFee.Sign() > 0 {
			if err = uniswapv3.MulDivV2(&step.feeAmount, &communityFee, COMMUNITY_FEE_DENOMINATOR, &delta,
				nil); err != nil {
				return err
			}
			step.feeAmount.Sub(&step.feeAmount, &delta)
			communityFeeAmount.Add(&communityFeeAmount, &delta)
		}

		if currentPrice.Eq(&step.nextTickPrice) {
			tickData, err := p.ticks.GetTick(nextTick)
			if err != nil {
				return err
			}

			if zeroToOne {
				currentTick = step.nextTick - 1
				initializedTick = step.nextTick - 1
			} else {
				currentTick = step.nextTick
				initializedTick = step.nextTick
			}
			if err = addDelta(currentLiquidity, tickData.LiquidityNet, zeroToOne); err != nil {
				return err
			}
		} else if !currentPrice.Eq(&step.stepSqrtPrice) {
			currentTickInt, err := uniswapv3.GetTickAtSqrtRatio(currentPrice)
			if err != nil {
				return err
			}
			currentTick = int32(currentTickInt)

			break
		}

		if amountRequired.IsZero() || currentPrice.Eq(limitSqrtPrice) {
			break
		}
	}

	o.remaining.Set(amountRequired)
	o.fee.Add(&communityFeeAmount, &pluginFeeAmount)
	o.tick = currentTick
	o.gas = BaseGas + ticksCrossed*CrossInitTickGas
	return nil
}

// movePriceTowardsTarget prices one swap step into resultPrice, input, output and feeAmount, which
// must not alias the inputs.
func movePriceTowardsTarget(zeroToOne bool, currentPrice, targetPrice, liquidity, amountAvailable *uint256.Int,
	fee uint64, resultPrice, input, output, feeAmount *uint256.Int) error {
	getInputTokenAmount, getOutputTokenAmount := getInputTokenDelta10, getOutputTokenDelta10
	if zeroToOne {
		getInputTokenAmount, getOutputTokenAmount = getInputTokenDelta01, getOutputTokenDelta01
	}

	var feeDenoMinusFee uint256.Int
	feeDenoMinusFee.SubUint64(FEE_DENOMINATOR, fee)
	if amountAvailable.Sign() >= 0 {
		var amountAvailableAfterFee uint256.Int
		if _, overflow := amountAvailableAfterFee.MulDivOverflow(amountAvailable, &feeDenoMinusFee,
			FEE_DENOMINATOR); overflow {
			return ErrOverflow
		}

		if err := getInputTokenAmount(input, targetPrice, currentPrice, liquidity); err != nil {
			return err
		}

		if !amountAvailableAfterFee.Lt(input) {
			resultPrice.Set(targetPrice)
			if err := v3Utils.MulDivRoundingUpV2(input, feeAmount.SetUint64(fee), &feeDenoMinusFee,
				feeAmount); err != nil {
				return err
			}
		} else {
			if err := getNewPriceAfterInput(resultPrice, currentPrice, liquidity, &amountAvailableAfterFee,
				zeroToOne); err != nil {
				return err
			}

			if targetPrice.Eq(resultPrice) {
				return fmt.Errorf("target price should not equal result price")
			}

			if err := getInputTokenAmount(input, resultPrice, currentPrice, liquidity); err != nil {
				return err
			}

			feeAmount.Sub(amountAvailable, input)
		}

		return getOutputTokenAmount(output, resultPrice, currentPrice, liquidity)
	}

	if err := getOutputTokenAmount(output, targetPrice, currentPrice, liquidity); err != nil {
		return err
	}

	var available uint256.Int
	if available.Neg(amountAvailable).Sign() < 0 {
		return ErrInvalidAmountRequired
	}

	if !available.Lt(output) {
		resultPrice.Set(targetPrice)
	} else {
		if err := getNewPriceAfterOutput(resultPrice, currentPrice, liquidity, &available, zeroToOne); err != nil {
			return err
		}

		if !targetPrice.Eq(resultPrice) {
			if err := getOutputTokenAmount(output, resultPrice, currentPrice, liquidity); err != nil {
				return err
			}
		}

		if output.Gt(&available) {
			output.Set(&available)
		}
	}

	if err := getInputTokenAmount(input, resultPrice, currentPrice, liquidity); err != nil {
		return err
	}

	return v3Utils.MulDivRoundingUpV2(input, feeAmount.SetUint64(fee), &feeDenoMinusFee, feeAmount)
}
