package pools

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	FullRangePoolSwapState struct {
		SqrtRatio *uint256.Int `json:"sqrtRatio"`
	}

	FullRangePoolState struct {
		*FullRangePoolSwapState
		Liquidity *uint256.Int `json:"liquidity"`
	}

	FullRangePool struct {
		key *FullRangePoolKey
		*FullRangePoolState
	}
)

func (p *FullRangePool) GetKey() IPoolKey {
	return p.key
}

func (p *FullRangePool) GetState() any {
	return p.FullRangePoolState
}

func (p *FullRangePool) CloneState() any {
	cloned := *p
	cloned.key = p.key.CloneState()
	clonedFullRangePoolState := *p.FullRangePoolState
	cloned.FullRangePoolState = &clonedFullRangePoolState
	return &cloned
}

func (p *FullRangePool) SetSwapState(state quoting.SwapState) {
	p.FullRangePoolSwapState = state.(*FullRangePoolSwapState)
}

func (p *FullRangePool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithLimitAndOverride(amount, isToken1, nil, nil)
}

func (p *FullRangePool) quoteWithLimitAndOverride(amount *uint256.Int, isToken1 bool, sqrtRatioLimit *uint256.Int, overrideState *FullRangePoolSwapState) (*quoting.Quote, error) {
	var state *FullRangePoolSwapState
	if overrideState == nil {
		state = p.FullRangePoolSwapState
	} else {
		state = overrideState
	}

	sqrtRatio := state.SqrtRatio

	isIncreasing := math.IsPriceIncreasing(amount, isToken1)

	if sqrtRatioLimit == nil {
		if isIncreasing {
			sqrtRatioLimit = math.MaxSqrtRatio
		} else {
			sqrtRatioLimit = math.MinSqrtRatio
		}
	}

	step, err := math.ComputeStep(
		sqrtRatio,
		p.Liquidity,
		sqrtRatioLimit,
		amount,
		isToken1,
		p.key.Config.Fee,
	)
	if err != nil {
		return nil, fmt.Errorf("swap step computation: %w", err)
	}

	sqrtRatio = step.SqrtRatioNext

	return &quoting.Quote{
		ConsumedAmount:   step.ConsumedAmount,
		CalculatedAmount: step.CalculatedAmount,
		FeesPaid:         step.FeeAmount,
		Gas:              quoting.BaseGasFullRangeSwap,
		SwapInfo: quoting.SwapInfo{
			SkipAhead:           0,
			IsToken1:            isToken1,
			SwapStateAfter:      NewFullRangePoolSwapState(sqrtRatio),
			TickSpacingsCrossed: 0,
		},
	}, nil
}

func NewFullRangePoolSwapState(sqrtRatio *uint256.Int) *FullRangePoolSwapState {
	return &FullRangePoolSwapState{
		SqrtRatio: sqrtRatio,
	}
}

func NewFullRangePoolState(swapState *FullRangePoolSwapState, liquidity *uint256.Int) *FullRangePoolState {
	return &FullRangePoolState{
		FullRangePoolSwapState: swapState,
		Liquidity:              liquidity,
	}
}

func NewFullRangePool(key *FullRangePoolKey, state *FullRangePoolState) *FullRangePool {
	return &FullRangePool{
		key:                key,
		FullRangePoolState: state,
	}
}
