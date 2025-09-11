package erc4626

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

		prefetchAmount := PrefetchAmounts[i]

		// Calculate multiplicative distance
		if amount.Gt(prefetchAmount) {
			diff.Div(amount, prefetchAmount)
		} else {
			diff.Div(prefetchAmount, amount)
		}

		if diff.Eq(u256.U1) {
			bestId = i
			break
		}

		diff.Sub(diff, u256.U1)

		if bestId == -1 || diff.Lt(bestDiff) {
			bestId = i
			bestDiff.Set(diff)
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
