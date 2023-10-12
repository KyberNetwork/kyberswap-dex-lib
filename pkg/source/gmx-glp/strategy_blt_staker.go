package gmxglp

import "math/big"

type StrategyBltStaker struct {
	Address              string
	EstimatedTotalAssets *big.Int `json:"estimatedTotalAssets"`
}

func NewStrategyBltStaker(address string, estimatedTotalAssets *big.Int) *StrategyBltStaker {
	return &StrategyBltStaker{
		Address:              address,
		EstimatedTotalAssets: estimatedTotalAssets,
	}
}

func (s *StrategyBltStaker) Withdraw(amountNeeded *big.Int) (*big.Int, *big.Int) {
	return s.liquidatePosition(amountNeeded)
}

func (s *StrategyBltStaker) liquidatePosition(amountNeeded *big.Int) (*big.Int, *big.Int) {
	wantBal := new(big.Int).Set(s.EstimatedTotalAssets)
	if amountNeeded.Cmp(wantBal) > 0 {
		withdrawBal := new(big.Int).Set(s.EstimatedTotalAssets)
		liquidatedAmount := new(big.Int).Set(amountNeeded)
		if liquidatedAmount.Cmp(withdrawBal) < 0 {
			liquidatedAmount = new(big.Int).Set(withdrawBal)
		}
		loss := new(big.Int).Sub(amountNeeded, liquidatedAmount)
		return liquidatedAmount, loss
	}

	return amountNeeded, big.NewInt(0)
}
