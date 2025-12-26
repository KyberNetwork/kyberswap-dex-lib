package tessera

import (
	"github.com/holiman/uint256"
)

func GetClosestRate(amountIn *uint256.Int, prefetches []PrefetchRate) (*uint256.Int, error) {
	if amountIn == nil || len(prefetches) == 0 {
		return nil, ErrInvalidRate
	}

	for i := range prefetches {
		if prefetches[i].Rate == nil && prefetches[i].AmountIn != nil {
			if amountIn.Cmp(prefetches[i].AmountIn) >= 0 {
				return nil, ErrSwapReverted
			}
		}
	}

	lowerIdx := -1
	for i := len(prefetches) - 1; i >= 0; i-- {
		if prefetches[i].Rate != nil && prefetches[i].AmountIn.Cmp(amountIn) <= 0 {
			lowerIdx = i
			break
		}
	}

	upperIdx := -1
	for i := 0; i < len(prefetches); i++ {
		if prefetches[i].Rate != nil && prefetches[i].AmountIn.Cmp(amountIn) >= 0 {
			upperIdx = i
			break
		}
	}

	if lowerIdx >= 0 && prefetches[lowerIdx].AmountIn.Eq(amountIn) {
		return prefetches[lowerIdx].Rate.Clone(), nil
	}
	if upperIdx >= 0 && prefetches[upperIdx].AmountIn.Eq(amountIn) {
		return prefetches[upperIdx].Rate.Clone(), nil
	}

	// Interpolation between bounds
	if lowerIdx >= 0 && upperIdx >= 0 {
		lowerRate := prefetches[lowerIdx].Rate
		upperRate := prefetches[upperIdx].Rate
		lowerAmount := prefetches[lowerIdx].AmountIn
		upperAmount := prefetches[upperIdx].AmountIn

		diffAmount := new(uint256.Int).Sub(upperAmount, lowerAmount)
		amountDiff := new(uint256.Int).Sub(amountIn, lowerAmount)

		var amountOut = new(uint256.Int)
		if upperRate.Cmp(lowerRate) >= 0 {
			diffRate := new(uint256.Int).Sub(upperRate, lowerRate)
			amountOut.MulDivOverflow(diffRate, amountDiff, diffAmount)
			return amountOut.Add(lowerRate, amountOut), nil
		} else {
			diffRate := new(uint256.Int).Sub(lowerRate, upperRate)
			amountOut.MulDivOverflow(diffRate, amountDiff, diffAmount)
			return amountOut.Sub(lowerRate, amountOut), nil
		}
	}

	if lowerIdx >= 0 {
		amountOut := amountIn.Clone()
		amountOut.MulDivOverflow(amountOut, prefetches[lowerIdx].Rate, prefetches[lowerIdx].AmountIn)
		return amountOut, nil
	}

	if upperIdx >= 0 {
		amountOut := amountIn.Clone()
		amountOut.MulDivOverflow(amountOut, prefetches[upperIdx].Rate, prefetches[upperIdx].AmountIn)
		return amountOut, nil
	}

	return nil, ErrInvalidRate
}
