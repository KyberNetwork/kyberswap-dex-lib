package erc4626

import (
	"github.com/holiman/uint256"
)

func GetClosestRate(rates []*uint256.Int, amount *uint256.Int) (*uint256.Int, error) {
	if len(rates) == 0 {
		return nil, ErrInvalidRate
	}

	bestId := -1
	bestDiff := new(uint256.Int)
	diff := new(uint256.Int)

	for i, rate := range rates {
		if rate == nil {
			continue
		}

		if PrefetchAmounts[i].Gt(amount) {
			diff.Sub(PrefetchAmounts[i], amount)
		} else {
			diff.Sub(amount, PrefetchAmounts[i])
		}

		if diff.IsZero() {
			bestId = i
			break
		}

		if bestId == -1 || diff.Lt(bestDiff) {
			bestId = i
			bestDiff, diff = diff, bestDiff
		}
	}

	if bestId == -1 {
		return nil, ErrInvalidRate
	}

	rate := rates[bestId]
	if rate.IsZero() {
		return nil, ErrInvalidRate
	}

	amount.MulDivOverflow(amount, rate, PrefetchAmounts[bestId])
	return amount, nil
}
