package pools

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	StableswapPoolSwapState struct {
		SqrtRatio *uint256.Int `json:"sqrtRatio"`
	}

	StableswapPoolState struct {
		*StableswapPoolSwapState
		Liquidity *uint256.Int `json:"liquidity"`
	}

	StableswapPool struct {
		key        *StableswapPoolKey
		lowerPrice uint256.Int
		upperPrice uint256.Int
		*StableswapPoolState
	}
)

func (p *StableswapPool) GetKey() IPoolKey {
	return p.key
}

func (p *StableswapPool) GetState() any {
	return p.StableswapPoolState
}

func (p *StableswapPool) CloneState() any {
	cloned := *p
	cloned.key = p.key.CloneState()
	clonedStableswapPoolState := *p.StableswapPoolState
	cloned.StableswapPoolState = &clonedStableswapPoolState
	return &cloned
}

func (p *StableswapPool) SetSwapState(state quoting.SwapState) {
	p.StableswapPoolSwapState = state.(*StableswapPoolSwapState)
}

func (p *StableswapPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	sqrtRatio := p.SqrtRatio.Clone()

	isIncreasing := math.IsPriceIncreasing(amount, isToken1)

	sqrtRatioLimit := lo.Ternary(isIncreasing, math.MaxSqrtRatio, math.MinSqrtRatio)

	var calculatedAmount, feesPaid uint256.Int
	amountRemaining := amount.Clone()
	var initializedTicksCrossed uint32 = 0

	for !amountRemaining.IsZero() && !sqrtRatio.Eq(sqrtRatioLimit) {
		stepLiquidity := p.Liquidity
		inRange := sqrtRatio.Lt(&p.upperPrice) && sqrtRatio.Gt(&p.lowerPrice)

		var nextTickSqrtRatio *uint256.Int
		if inRange {
			if isIncreasing {
				nextTickSqrtRatio = &p.upperPrice
			} else {
				nextTickSqrtRatio = &p.lowerPrice
			}
		} else {
			stepLiquidity = new(uint256.Int)

			if sqrtRatio.Lt(&p.lowerPrice) || sqrtRatio.Eq(&p.lowerPrice) {
				if isIncreasing {
					nextTickSqrtRatio = &p.lowerPrice
				}
			} else {
				if !isIncreasing {
					nextTickSqrtRatio = &p.upperPrice
				}
			}
		}

		stepSqrtRatioLimit := nextTickSqrtRatio
		if nextTickSqrtRatio == nil || nextTickSqrtRatio.Lt(sqrtRatioLimit) != isIncreasing {
			stepSqrtRatioLimit = sqrtRatioLimit
		}

		step, err := math.ComputeStep(
			sqrtRatio,
			stepLiquidity,
			stepSqrtRatioLimit,
			amountRemaining,
			isToken1,
			p.key.Config.Fee,
		)
		if err != nil {
			return nil, fmt.Errorf("swap step computation: %w", err)
		}

		amountRemaining.Sub(amountRemaining, step.ConsumedAmount)
		calculatedAmount.Add(&calculatedAmount, step.CalculatedAmount)
		feesPaid.Add(&feesPaid, step.FeeAmount)
		sqrtRatio = step.SqrtRatioNext

		if nextTickSqrtRatio != nil && nextTickSqrtRatio.Eq(sqrtRatio) {
			initializedTicksCrossed++
		}
	}

	return &quoting.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, amountRemaining),
		CalculatedAmount: &calculatedAmount,
		FeesPaid:         &feesPaid,
		Gas:              quoting.BaseGasStableswapSwap + int64(initializedTicksCrossed)*quoting.GasInitializedTickCrossed,
		SwapInfo: quoting.SwapInfo{
			SkipAhead:  0,
			IsToken1:   isToken1,
			PriceLimit: math.FixedSqrtRatioToFloat(sqrtRatioLimit, isIncreasing),
			SwapStateAfter: &StableswapPoolSwapState{
				sqrtRatio,
			},
			TickSpacingsCrossed: 0,
		},
	}, nil
}

func NewStableswapPool(key *StableswapPoolKey, state *StableswapPoolState) *StableswapPool {
	p := &StableswapPool{
		key:                 key,
		StableswapPoolState: state,
	}
	setStableswapBounds(p, key.Config.TypeConfig.CenterTick, key.Config.TypeConfig.AmplificationFactor)
	return p
}

func setStableswapBounds(pool *StableswapPool, centerTick int32, amplification uint8) {
	lower, upper := activeRange(centerTick, amplification)
	pool.lowerPrice.Set(math.ToSqrtRatio(lower))
	pool.upperPrice.Set(math.ToSqrtRatio(upper))
}

func activeRange(centerTick int32, amplification uint8) (int32, int32) {
	width := math.MaxTick >> amplification
	lower := centerTick - width
	if lower < math.MinTick {
		lower = math.MinTick
	}
	upper := centerTick + width
	if upper > math.MaxTick {
		upper = math.MaxTick
	}
	return lower, upper
}
