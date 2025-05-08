package pools

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
)

func (s *BasePoolState) CalcBalances() ([]big.Int, error) {
	stateSqrtRatio := s.SqrtRatio

	balances := make([]big.Int, 2)
	liquidity := new(big.Int)
	sqrtRatio := new(big.Int).Set(math.MinSqrtRatio)
	minAmount1SqrtRatio, maxAmount0SqrtRatio := new(big.Int), new(big.Int)

	for _, tick := range s.SortedTicks {
		tickSqrtRatio := math.ToSqrtRatio(tick.Number)
		if stateSqrtRatio.Cmp(tickSqrtRatio) > 0 {
			minAmount1SqrtRatio.Set(tickSqrtRatio)
		} else {
			minAmount1SqrtRatio.Set(stateSqrtRatio)
		}

		if stateSqrtRatio.Cmp(sqrtRatio) > 0 {
			maxAmount0SqrtRatio.Set(stateSqrtRatio)
		} else {
			maxAmount0SqrtRatio.Set(sqrtRatio)
		}

		if sqrtRatio.Cmp(minAmount1SqrtRatio) < 0 {
			amount1Delta, err := math.Amount1Delta(sqrtRatio, minAmount1SqrtRatio, liquidity, false)
			if err != nil {
				return nil, fmt.Errorf("computing amount1 delta: %w", err)
			}
			balances[1].Add(&balances[1], amount1Delta)
		}
		if maxAmount0SqrtRatio.Cmp(tickSqrtRatio) < 0 {
			amount0Delta, err := math.Amount0Delta(maxAmount0SqrtRatio, tickSqrtRatio, liquidity, false)
			if err != nil {
				return nil, fmt.Errorf("computing amount0 delta: %w", err)
			}
			balances[0].Add(&balances[0], amount0Delta)
		}

		sqrtRatio.Set(tickSqrtRatio)
		liquidity.Add(liquidity, tick.LiquidityDelta)
	}

	return balances, nil
}

func (s *FullRangePoolState) CalcBalances() ([]big.Int, error) {
	tvl0, err := math.Amount0Delta(s.SqrtRatio, math.MaxSqrtRatio, s.Liquidity, false)
	if err != nil {
		return nil, fmt.Errorf("computing amount0 delta: %w", err)
	}

	tvl1, err := math.Amount1Delta(math.MinSqrtRatio, s.SqrtRatio, s.Liquidity, false)
	if err != nil {
		return nil, fmt.Errorf("computing amount1 delta: %w", err)
	}

	return []big.Int{*tvl0, *tvl1}, nil
}
