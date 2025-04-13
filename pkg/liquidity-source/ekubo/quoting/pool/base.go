package pool

import (
	"fmt"
	"math/big"

	math2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

func NewBasePool(poolKey quoting2.PoolKey, poolState quoting2.PoolState) BasePool {
	return BasePool{
		sqrtRatio:       new(big.Int).Set(poolState.SqrtRatio),
		liquidity:       new(big.Int).Set(poolState.Liquidity),
		activeTickIndex: quoting2.NearestInitializedTickIndex(poolState.Ticks, poolState.ActiveTick),
		sortedTicks:     poolState.Ticks,
		tickBounds:      poolState.TickBounds,
		poolKey:         poolKey,
	}
}

type BasePool struct {
	sqrtRatio       *big.Int
	liquidity       *big.Int
	activeTickIndex int
	sortedTicks     []quoting2.Tick
	tickBounds      [2]int32

	poolKey quoting2.PoolKey
}

type nextInitializedTick struct {
	*quoting2.Tick
	Index     int
	SqrtRatio *big.Int
}

func (p *BasePool) SetState(state quoting2.StateAfter) {
	p.sqrtRatio = new(big.Int).Set(state.SqrtRatio)
	p.liquidity = new(big.Int).Set(state.Liquidity)
	p.activeTickIndex = state.ActiveTickIndex
}

func (p *BasePool) Quote(amount *big.Int, isToken1 bool) (*quoting2.Quote, error) {
	sqrtRatio := new(big.Int).Set(p.sqrtRatio)
	liquidity := new(big.Int).Set(p.liquidity)
	activeTickIndex := p.activeTickIndex

	if amount.Sign() == 0 {
		return &quoting2.Quote{
			ConsumedAmount:   new(big.Int),
			CalculatedAmount: new(big.Int),
			FeesPaid:         new(big.Int),
			Gas:              0,
			SwapInfo: quoting2.SwapInfo{
				StateAfter: quoting2.StateAfter{
					SqrtRatio:       sqrtRatio,
					Liquidity:       liquidity,
					ActiveTickIndex: activeTickIndex,
				},
				SkipAhead: 0,
			},
		}, nil
	}

	isIncreasing := math2.IsPriceIncreasing(amount, isToken1)

	var sqrtRatioLimit *big.Int
	if isIncreasing {
		sqrtRatioLimit = math2.MaxSqrtRatio
	} else {
		sqrtRatioLimit = math2.MinSqrtRatio
	}

	calculatedAmount := new(big.Int)
	feesPaid := new(big.Int)
	var initializedTicksCrossed uint32 = 0
	amountRemaining := new(big.Int).Set(amount)

	startingSqrtRatio := new(big.Int).Set(sqrtRatio)

	for amountRemaining.Sign() != 0 && sqrtRatio.Cmp(sqrtRatioLimit) != 0 {
		var nextInitTick *nextInitializedTick
		if isIncreasing {
			if activeTickIndex != quoting2.InvalidTickIndex {
				nextTickIndex := activeTickIndex + 1
				if nextTickIndex < len(p.sortedTicks) {
					tick := &p.sortedTicks[nextTickIndex]
					nextInitTick = &nextInitializedTick{
						Tick:      tick,
						Index:     nextTickIndex,
						SqrtRatio: math2.ToSqrtRatio(tick.Number),
					}
				}
			} else if len(p.sortedTicks) > 0 {
				tick := &p.sortedTicks[0]
				nextInitTick = &nextInitializedTick{
					Tick:      tick,
					Index:     0,
					SqrtRatio: math2.ToSqrtRatio(tick.Number),
				}
			}
		} else if activeTickIndex != quoting2.InvalidTickIndex {
			tick := &p.sortedTicks[activeTickIndex]
			nextInitTick = &nextInitializedTick{
				Tick:      tick,
				Index:     activeTickIndex,
				SqrtRatio: math2.ToSqrtRatio(tick.Number),
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

		step, err := math2.ComputeStep(
			sqrtRatio,
			liquidity,
			stepSqrtRatioLimit,
			amountRemaining,
			isToken1,
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
					activeTickIndex = quoting2.InvalidTickIndex
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
				activeTickIndex = quoting2.InvalidTickIndex
			}
		}
	}

	tickSpacingsCrossed := math2.ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, sqrtRatio, p.poolKey.Config.TickSpacing)

	var skipAhead uint32
	if initializedTicksCrossed != 0 {
		skipAhead = tickSpacingsCrossed / initializedTicksCrossed
	}

	return &quoting2.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, amountRemaining),
		CalculatedAmount: calculatedAmount,
		FeesPaid:         feesPaid,
		Gas:              quoting2.BaseGasCostOfOneSwap + int64(initializedTicksCrossed)*quoting2.GasCostOfOneInitializedTickCrossed + int64(tickSpacingsCrossed)*quoting2.GasCostOfOneTickSpacingCrossed,
		SwapInfo: quoting2.SwapInfo{
			SkipAhead: skipAhead,
			StateAfter: quoting2.StateAfter{
				SqrtRatio:       sqrtRatio,
				Liquidity:       liquidity,
				ActiveTickIndex: activeTickIndex,
			},
		},
	}, nil
}

func (p *BasePool) GetKey() *quoting2.PoolKey {
	return &p.poolKey
}
