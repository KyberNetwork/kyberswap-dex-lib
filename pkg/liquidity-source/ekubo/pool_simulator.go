package ekubo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	ekubo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	hook       ekubo.IHook
	hookConfig ekubo.HooksConfig

	sqrtRatio       *big.Int
	liquidity       *big.Int
	activeTickIndex int
	sortedTicks     []quoting.Tick

	poolKey quoting.PoolKey
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

	var hook ekubo.IHook
	switch staticExtra.ExtensionType {
	case Oracle:
		hook = ekubo.NewOracleHook()
	default:
		hook = ekubo.NewNoOpHook()
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		hook:            hook,
		hookConfig:      ExtensionConfigs[staticExtra.ExtensionType],
		sqrtRatio:       extra.SqrtRatio,
		liquidity:       extra.Liquidity,
		activeTickIndex: quoting.NearestInitializedTickIndex(extra.Ticks, extra.ActiveTick),
		sortedTicks:     extra.Ticks,
		poolKey:         staticExtra.PoolKey,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	poolSwapParams := &ekubo.PoolSwapParams{
		PoolKey:        p.poolKey,
		Amount:         params.TokenAmountIn.Amount,
		IsToken1:       strings.EqualFold(p.poolKey.Token1.Hex(), params.TokenAmountIn.Token),
		SqrtRatioLimit: p.sqrtRatio,
	}

	totalGas := quoting.BaseGasCostOfOneSwap

	if p.hookConfig.ShouldCallBeforeSwap {
		gas, err := p.hook.OnBeforeSwap(poolSwapParams)
		if err != nil {
			return nil, err
		}
		totalGas += gas
	}

	quote, err := p.swap(poolSwapParams)
	if err != nil {
		return nil, err
	}

	totalGas += uint64(quote.InitializedTicksCrossed)*quoting.GasCostOfOneInitializedTickCrossed +
		uint64(quote.TickSpacingsCrossed)*quoting.GasCostOfOneTickSpacingCrossed

	if p.hookConfig.ShouldCallAfterSwap {
		gas, err := p.hook.OnAfterSwap(quote)
		if err != nil {
			return nil, err
		}
		totalGas += gas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: quote.CalculatedAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: quote.FeesPaid,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: new(big.Int).Sub(params.TokenAmountIn.Amount, quote.ConsumedAmount),
		},
		Gas: int64(totalGas),
		SwapInfo: &SwapInfo{
			SkipAhead:       quote.SkipAhead,
			SqrtRatio:       quote.SqrtRatio,
			Liquidity:       quote.Liquidity,
			ActiveTickIndex: quote.ActiveTickIndex,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	poolSwapParams := &ekubo.PoolSwapParams{
		PoolKey:        p.poolKey,
		Amount:         new(big.Int).Neg(params.TokenAmountOut.Amount),
		IsToken1:       strings.EqualFold(p.poolKey.Token1.Hex(), params.TokenAmountOut.Token),
		SqrtRatioLimit: p.sqrtRatio,
	}

	totalGas := quoting.BaseGasCostOfOneSwap

	if p.hookConfig.ShouldCallBeforeSwap {
		gas, err := p.hook.OnBeforeSwap(poolSwapParams)
		if err != nil {
			return nil, err
		}
		totalGas += gas
	}

	quote, err := p.swap(poolSwapParams)
	if err != nil {
		return nil, err
	}

	totalGas += uint64(quote.InitializedTicksCrossed)*quoting.GasCostOfOneInitializedTickCrossed +
		uint64(quote.TickSpacingsCrossed)*quoting.GasCostOfOneTickSpacingCrossed

	if p.hookConfig.ShouldCallAfterSwap {
		gas, err := p.hook.OnAfterSwap(quote)
		if err != nil {
			return nil, err
		}
		totalGas += gas
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenIn,
			Amount: quote.CalculatedAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountOut.Token,
			Amount: quote.FeesPaid,
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenAmountOut.Token,
			Amount: new(big.Int).Add(params.TokenAmountOut.Amount, quote.ConsumedAmount),
		},
		Gas: int64(totalGas),
		SwapInfo: &SwapInfo{
			SkipAhead:       quote.SkipAhead,
			SqrtRatio:       quote.SqrtRatio,
			Liquidity:       quote.Liquidity,
			ActiveTickIndex: quote.ActiveTickIndex,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState := params.SwapInfo.(*SwapInfo)
	p.sqrtRatio = new(big.Int).Set(newState.SqrtRatio)
	p.liquidity = new(big.Int).Set(newState.Liquidity)
	p.activeTickIndex = newState.ActiveTickIndex
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}

func (p *PoolSimulator) swap(params *ekubo.PoolSwapParams) (*ekubo.SwapResult, error) {
	sqrtRatio := new(big.Int).Set(p.sqrtRatio)
	liquidity := new(big.Int).Set(p.liquidity)
	activeTickIndex := p.activeTickIndex

	if params.Amount.Sign() == 0 {
		return nil, ErrZeroSwapAmount
	}

	isIncreasing := math.IsPriceIncreasing(params.Amount, params.IsToken1)

	var sqrtRatioLimit *big.Int
	if isIncreasing {
		sqrtRatioLimit = math.MaxSqrtRatio
	} else {
		sqrtRatioLimit = math.MinSqrtRatio
	}

	calculatedAmount := new(big.Int)
	feesPaid := new(big.Int)
	var initializedTicksCrossed uint32 = 0
	amountRemaining := new(big.Int).Set(params.Amount)

	startingSqrtRatio := new(big.Int).Set(sqrtRatio)

	for amountRemaining.Sign() != 0 && sqrtRatio.Cmp(sqrtRatioLimit) != 0 {
		var nextInitTick *nextInitializedTick
		if isIncreasing {
			if activeTickIndex != quoting.InvalidTickIndex {
				nextTickIndex := activeTickIndex + 1
				if nextTickIndex < len(p.sortedTicks) {
					tick := &p.sortedTicks[nextTickIndex]
					nextInitTick = &nextInitializedTick{
						Tick:      tick,
						Index:     nextTickIndex,
						SqrtRatio: math.ToSqrtRatio(tick.Number),
					}
				}
			} else if len(p.sortedTicks) > 0 {
				tick := &p.sortedTicks[0]
				nextInitTick = &nextInitializedTick{
					Tick:      tick,
					Index:     0,
					SqrtRatio: math.ToSqrtRatio(tick.Number),
				}
			}
		} else if activeTickIndex != quoting.InvalidTickIndex {
			tick := &p.sortedTicks[activeTickIndex]
			nextInitTick = &nextInitializedTick{
				Tick:      tick,
				Index:     activeTickIndex,
				SqrtRatio: math.ToSqrtRatio(tick.Number),
			}
		}

		var stepSqrtRatioLimit *big.Int
		if nextInitTick == nil {
			stepSqrtRatioLimit = new(big.Int).Set(sqrtRatioLimit)
		} else {
			nextRatio := new(big.Int).Set(nextInitTick.SqrtRatio)
			if (nextRatio.Cmp(sqrtRatioLimit) == -1) == isIncreasing {
				stepSqrtRatioLimit = nextRatio
			} else {
				stepSqrtRatioLimit = new(big.Int).Set(sqrtRatioLimit)
			}
		}

		step, err := math.ComputeStep(
			sqrtRatio,
			liquidity,
			stepSqrtRatioLimit,
			amountRemaining,
			params.IsToken1,
			p.poolKey.Config.Fee,
		)
		if err != nil {
			return nil, fmt.Errorf("swap step computation: %w", err)
		}

		amountRemaining.Sub(amountRemaining, step.ConsumedAmount)
		calculatedAmount.Add(calculatedAmount, step.CalculatedAmount)
		feesPaid.Add(feesPaid, step.FeeAmount)
		sqrtRatio = step.SqrtRatioNext

		if nextInitTick != nil {
			tickIndex := nextInitTick.Index
			if sqrtRatio.Cmp(nextInitTick.SqrtRatio) == 0 {
				if isIncreasing {
					activeTickIndex = tickIndex
				} else if tickIndex != 0 {
					activeTickIndex = tickIndex - 1
				} else {
					activeTickIndex = quoting.InvalidTickIndex
				}

				initializedTicksCrossed += 1

				liquidityDelta := nextInitTick.LiquidityDelta
				liquidityDeltaAbs := new(big.Int).Abs(liquidityDelta)
				if (liquidityDelta.Sign() == 1) == isIncreasing {
					liquidity.Add(liquidity, liquidityDeltaAbs)
				} else {
					liquidity.Sub(liquidity, liquidityDeltaAbs)
				}
			}
		} else {
			if isIncreasing && len(p.sortedTicks) > 0 {
				activeTickIndex = len(p.sortedTicks) - 1
			} else {
				activeTickIndex = quoting.InvalidTickIndex
			}
		}
	}

	tickSpacingsCrossed := math.ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, sqrtRatio, p.poolKey.Config.TickSpacing)

	var skipAhead uint32
	if initializedTicksCrossed != 0 {
		skipAhead = tickSpacingsCrossed / initializedTicksCrossed
	}

	return &ekubo.SwapResult{
		ConsumedAmount:   amountRemaining.Sub(params.Amount, amountRemaining),
		CalculatedAmount: calculatedAmount,
		FeesPaid:         feesPaid,
		SkipAhead:        skipAhead,
		SqrtRatio:        sqrtRatio,
		Liquidity:        liquidity,
		ActiveTickIndex:  activeTickIndex,

		InitializedTicksCrossed: initializedTicksCrossed,
		TickSpacingsCrossed:     tickSpacingsCrossed,
	}, nil
}
