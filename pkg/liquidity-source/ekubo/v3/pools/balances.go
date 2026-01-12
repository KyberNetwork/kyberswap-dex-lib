package pools

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func (s *BasePoolState) CalcBalances() ([]uint256.Int, error) {
	stateSqrtRatio := s.SqrtRatio

	balances := make([]uint256.Int, 2)
	var liquidity, sqrtRatio, minAmount1SqrtRatio, maxAmount0SqrtRatio uint256.Int
	sqrtRatio.Set(math.MinSqrtRatio)

	for _, tick := range s.SortedTicks {
		tickSqrtRatio := math.ToSqrtRatio(tick.Number)
		minAmount1SqrtRatio.Set(big256.Min(tickSqrtRatio, stateSqrtRatio))
		maxAmount0SqrtRatio.Set(big256.Max(stateSqrtRatio, &sqrtRatio))
		if sqrtRatio.Lt(&minAmount1SqrtRatio) {
			amount1Delta, err := math.Amount1Delta(&sqrtRatio, &minAmount1SqrtRatio, &liquidity, false)
			if err != nil {
				return nil, fmt.Errorf("computing amount1 delta: %w", err)
			}
			balances[1].Add(&balances[1], amount1Delta)
		}
		if maxAmount0SqrtRatio.Lt(tickSqrtRatio) {
			amount0Delta, err := math.Amount0Delta(&maxAmount0SqrtRatio, tickSqrtRatio, &liquidity, false)
			if err != nil {
				return nil, fmt.Errorf("computing amount0 delta: %w", err)
			}
			balances[0].Add(&balances[0], amount0Delta)
		}

		sqrtRatio.Set(tickSqrtRatio)
		liquidity.Add(&liquidity, (*uint256.Int)(tick.LiquidityDelta))
	}

	return balances, nil
}

func (s *FullRangePoolState) CalcBalances() ([]uint256.Int, error) {
	tvl0, err := math.Amount0Delta(s.SqrtRatio, math.MaxSqrtRatio, s.Liquidity, false)
	if err != nil {
		return nil, fmt.Errorf("computing amount0 delta: %w", err)
	}

	tvl1, err := math.Amount1Delta(math.MinSqrtRatio, s.SqrtRatio, s.Liquidity, false)
	if err != nil {
		return nil, fmt.Errorf("computing amount1 delta: %w", err)
	}

	return []uint256.Int{*tvl0, *tvl1}, nil
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
