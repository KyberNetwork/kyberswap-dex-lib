package mooniswap

import (
	"github.com/holiman/uint256"
)

func calcAmountOut(amount, srcBalance, dstBalance, fee, slippageFee *uint256.Int) *uint256.Int {
	if amount.IsZero() {
		return new(uint256.Int)
	}

	var feeAmount, taxedAmount uint256.Int
	feeAmount.MulDivOverflow(amount, fee, uFeeDenominator)
	taxedAmount.Sub(amount, &feeAmount)

	srcBalPlusTaxed := new(uint256.Int).Add(srcBalance, &taxedAmount)

	var res uint256.Int
	res.MulDivOverflow(&taxedAmount, dstBalance, srcBalPlusTaxed)

	var feeNum, feeDen, slipPart uint256.Int
	feeDen.Mul(uFeeDenominator, srcBalPlusTaxed)
	slipPart.Mul(slippageFee, &taxedAmount)
	feeNum.Sub(&feeDen, &slipPart)

	res.MulDivOverflow(&res, &feeNum, &feeDen)

	return &res
}
