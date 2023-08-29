package crowdswapv2

import (
	"math/big"
)

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	tokenWeightIn uint,
	tokenWeightOut uint,
	swapFee *big.Int,
) (res *big.Int, err error) {
	var amountInWithFee = new(big.Int).Mul(amountIn, new(big.Int).Sub(bOne, swapFee))
	if tokenWeightIn == tokenWeightOut {
		res = new(big.Int).Div(new(big.Int).Mul(reserveOut, amountInWithFee), new(big.Int).Add(new(big.Int).Mul(reserveIn, bOne), amountInWithFee))
		return res, nil
	}
	var baseN = new(big.Int).Add(new(big.Int).Mul(reserveIn, bOne), amountInWithFee)
	res, precision, err := Power(baseN, new(big.Int).Mul(reserveIn, bOne), tokenWeightIn, tokenWeightOut)
	if err != nil {
		return nil, err
	}
	var temp1 = new(big.Int).Mul(reserveOut, res)
	var temp2 = new(big.Int).Lsh(reserveOut, precision)
	res = new(big.Int).Div(new(big.Int).Sub(temp1, temp2), res)
	return res, nil
}
