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

func (p *StableswapPool) CloneSwapStateOnly() Pool {
	cloned := *p
	copiedStableswapPoolState := *p.StableswapPoolState
	cloned.StableswapPoolState = &copiedStableswapPoolState
	cloned.StableswapPoolSwapState = p.Clone()
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
	movedOutOfBoundary := false

	for !amountRemaining.IsZero() && !sqrtRatio.Eq(sqrtRatioLimit) {
		stepLiquidity := p.Liquidity
		inRange := sqrtRatio.Cmp(&p.upperPrice) <= 0 && sqrtRatio.Cmp(&p.lowerPrice) >= 0 && !movedOutOfBoundary

		var nextTickSqrtRatio *uint256.Int
		if inRange {
			if isIncreasing {
				nextTickSqrtRatio = &p.upperPrice
			} else {
				nextTickSqrtRatio = &p.lowerPrice
			}
		} else {
			stepLiquidity = new(uint256.Int)

			if !movedOutOfBoundary {
				if sqrtRatio.Lt(&p.lowerPrice) {
					if isIncreasing {
						nextTickSqrtRatio = &p.lowerPrice
					}
				} else if !isIncreasing {
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

		if nextTickSqrtRatio != nil && nextTickSqrtRatio.Eq(sqrtRatio) && (sqrtRatio.Eq(&p.upperPrice) && isIncreasing || sqrtRatio.Eq(&p.lowerPrice) && !isIncreasing) {
			movedOutOfBoundary = true
		}
	}

	return &quoting.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, amountRemaining),
		CalculatedAmount: &calculatedAmount,
		FeesPaid:         &feesPaid,
		Gas:              quoting.GasCostOfOneStableswapSwap,
		SwapInfo: quoting.SwapInfo{
			SkipAhead:           0,
			IsToken1:            isToken1,
			PriceLimit:          math.FixedSqrtRatioToFloat(sqrtRatioLimit, isIncreasing),
			SwapStateAfter:      NewStableswapPoolSwapState(sqrtRatio),
			TickSpacingsCrossed: 0,
		},
	}, nil
}

func (p *StableswapPool) CalcBalances() ([]uint256.Int, error) {
	tvl0, tvl1 := new(uint256.Int), new(uint256.Int)
	var err error

	if p.SqrtRatio.Lt(&p.upperPrice) {
		tvl0, err = math.Amount0Delta(p.SqrtRatio, &p.upperPrice, p.Liquidity, false)
		if err != nil {
			return nil, fmt.Errorf("computing amount0 delta: %w", err)
		}
	}

	if p.SqrtRatio.Gt(&p.lowerPrice) {
		tvl1, err = math.Amount1Delta(&p.lowerPrice, p.SqrtRatio, p.Liquidity, false)
		if err != nil {
			return nil, fmt.Errorf("computing amount1 delta: %w", err)
		}
	}

	return []uint256.Int{*tvl0, *tvl1}, nil
}

func (p *StableswapPool) ApplyEvent(event Event, data []byte, _ uint64) error {
	switch event {
	case EventSwapped:
		event, err := parseSwappedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.SqrtRatio = event.sqrtRatioAfter
		p.Liquidity = event.liquidityAfter
	case EventPositionUpdated:
		event, err := parsePositionUpdatedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.Liquidity.Add(p.Liquidity, (*uint256.Int)(event.liquidityDelta))
	default:
	}

	return nil
}

func (p *StableswapPool) NewBlock() {}

func (s *StableswapPoolSwapState) Clone() *StableswapPoolSwapState {
	return NewStableswapPoolSwapState(s.SqrtRatio.Clone())
}

func NewStableswapPoolSwapState(sqrtRatio *uint256.Int) *StableswapPoolSwapState {
	return &StableswapPoolSwapState{
		SqrtRatio: sqrtRatio,
	}
}

func NewStableswapPoolState(swapState *StableswapPoolSwapState, liquidity *uint256.Int) *StableswapPoolState {
	return &StableswapPoolState{
		StableswapPoolSwapState: swapState,
		Liquidity:               liquidity,
	}
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
