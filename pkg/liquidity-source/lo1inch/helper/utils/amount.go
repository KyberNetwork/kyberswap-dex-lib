package utils

import "github.com/holiman/uint256"

// CalcTakingAmount https://github.com/1inch/limit-order-protocol/blob/23d655844191dea7960a186652307604a1ed480a/contracts/libraries/AmountCalculatorLib.sol#L6
func CalcTakingAmount(swapMakerAmount, orderMakerAmount, orderTakerAmount *uint256.Int) *uint256.Int {
	amount := new(uint256.Int).Mul(swapMakerAmount, orderTakerAmount)
	amount.Add(amount, orderMakerAmount)
	amount.Sub(amount, uint256.NewInt(1))
	return amount.Div(amount, orderMakerAmount)
}

// CalcMakingAmount https://github.com/1inch/limit-order-protocol/blob/23d655844191dea7960a186652307604a1ed480a/contracts/libraries/AmountCalculatorLib.sol#L6
func CalcMakingAmount(swapTakerAmount, orderMakerAmount, orderTakerAmount *uint256.Int) *uint256.Int {
	amount := new(uint256.Int).Mul(swapTakerAmount, orderMakerAmount)
	return amount.Div(amount, orderTakerAmount)
}
