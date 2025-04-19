package pools

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type FullRangePoolSwapState struct {
	SqrtRatio *big.Int `json:"sqrtRatio"`
}

type FullRangePoolState struct {
	*FullRangePoolSwapState
	Liquidity *big.Int `json:"liquidity"`
}

type FullRangePool struct {
	key *PoolKey
	*FullRangePoolState
}

func NewFullRangePool(key *PoolKey, state *FullRangePoolState) *FullRangePool {
	return &FullRangePool{
		key,
		state,
	}
}

func (p *FullRangePool) GetKey() *PoolKey {
	return p.key
}

func (p *FullRangePool) GetState() any {
	return p.FullRangePoolState
}

func (p *FullRangePool) SetSwapState(state any) {
	p.FullRangePoolSwapState = state.(*FullRangePoolSwapState)
}

func (p *FullRangePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithLimitAndOverride(amount, isToken1, nil, nil)
}

func (p *FullRangePool) quoteWithLimitAndOverride(amount *big.Int, isToken1 bool, sqrtRatioLimit *big.Int, overrideState *FullRangePoolSwapState) (*quoting.Quote, error) {
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
		Gas:              quoting.BaseGasCostOfOneFullRangeSwap,
		SwapInfo: quoting.SwapInfo{
			SkipAhead: 0,
			IsToken1:  isToken1,
			SwapStateAfter: FullRangePoolSwapState{
				sqrtRatio,
			},
			TickSpacingsCrossed: 0,
		},
	}, nil
}
