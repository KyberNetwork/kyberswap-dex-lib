package erc4626

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

func GetClosestRate(rates []*uint256.Int, amount *uint256.Int, isExactOut bool) (*uint256.Int, error) {
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

		inAmt := amount.Clone()
		if isExactOut {
			// in case of exact out, calculate in amount so that we can calculate diff base on in amount.
			inAmt.MulDivOverflow(amount, prefetchAmount, rate)
		}

		// Calculate multiplicative distance
		if inAmt.Gt(prefetchAmount) {
			diff.Div(inAmt, prefetchAmount)
		} else {
			diff.Div(prefetchAmount, inAmt)
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

	// new result here to avoid modifying amount param
	result := uint256.NewInt(0)
	if isExactOut {
		// in = out * prefetchAmount / rate
		result.MulDivOverflow(amount, PrefetchAmounts[bestId], rate)
	} else {
		// out = in * rate / prefetchAmount
		result.MulDivOverflow(amount, rate, PrefetchAmounts[bestId])
	}

	return result, nil
}
