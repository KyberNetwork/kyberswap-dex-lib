package pool

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
)

func NewBasePool(poolKey quoting.PoolKey, poolState quoting.PoolState) BasePool {
	return BasePool{
		sqrtRatio:       new(big.Int).Set(poolState.SqrtRatio),
		liquidity:       new(big.Int).Set(poolState.Liquidity),
		activeTickIndex: quoting.NearestInitializedTickIndex(poolState.Ticks, poolState.ActiveTick),
		sortedTicks:     poolState.Ticks,
		tickBounds:      poolState.TickBounds,
		poolKey:         poolKey,
	}
}

type BasePool struct {
	sqrtRatio       *big.Int
	liquidity       *big.Int
	activeTickIndex int
	sortedTicks     []quoting.Tick
	tickBounds      [2]int32

	poolKey quoting.PoolKey
}

type nextInitializedTick struct {
	*quoting.Tick
	Index     int
	SqrtRatio *big.Int
}

func (p *BasePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	if amount.Sign() == 0 {
		return &quoting.Quote{
			ConsumedAmount:   new(big.Int),
			CalculatedAmount: new(big.Int),
			FeesPaid:         new(big.Int),
			Gas:              0,
		}, nil
	}

	isIncreasing := math.IsPriceIncreasing(amount, isToken1)

	sqrtRatio := new(big.Int).Set(p.sqrtRatio)
	liquidity := new(big.Int).Set(p.liquidity)
	activeTickIndex := p.activeTickIndex

	var sqrtRatioLimit *big.Int
	if isIncreasing {
		sqrtRatioLimit = math.MaxSqrtRatio
	} else {
		sqrtRatioLimit = math.MinSqrtRatio
	}

	calculatedAmount := new(big.Int)
	feesPaid := new(big.Int)
	var initializedTicksCrossed uint32 = 0
	amountRemaining := new(big.Int).Set(amount)

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

	return &quoting.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, amountRemaining),
		CalculatedAmount: calculatedAmount,
		FeesPaid:         feesPaid,
		Gas:              quoting.BaseGasCostOfOneSwap + int64(initializedTicksCrossed)*quoting.GasCostOfOneInitializedTickCrossed + int64(tickSpacingsCrossed)*quoting.GasCostOfOneTickSpacingCrossed,
		SkipAhead:        skipAhead,
	}, nil
}

func (p *BasePool) GetKey() *quoting.PoolKey {
	return &p.poolKey
}
