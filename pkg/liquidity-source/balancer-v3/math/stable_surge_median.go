package math

import (
	"slices"

	"github.com/holiman/uint256"
)

var StableSurgeMedian *stableSurgeMedian

type stableSurgeMedian struct{}

func (s *stableSurgeMedian) CalculateImbalance(balances []*uint256.Int) (*uint256.Int, error) {
	median, err := s.findMedian(balances)
	if err != nil {
		return nil, err
	}

	var totalBalance, totalDiff, diff uint256.Int
	for _, balance := range balances {
		totalBalance.Add(&totalBalance, balance)
		totalDiff.Add(&totalDiff, diff.Abs(diff.Sub(balance, median)))
	}

	return FixPoint.DivDown(&totalDiff, &totalBalance)
}

func (s *stableSurgeMedian) findMedian(balances []*uint256.Int) (*uint256.Int, error) {
	sortedBalances := slices.SortedFunc(slices.Values(balances), func(i, j *uint256.Int) int {
		return i.Cmp(j)
	})
	mid := len(sortedBalances) / 2

	if len(sortedBalances)%2 == 0 {
		var avg uint256.Int
		return avg.Div(avg.Add(sortedBalances[mid-1], sortedBalances[mid]), U2), nil
	}

	return sortedBalances[mid], nil
}
