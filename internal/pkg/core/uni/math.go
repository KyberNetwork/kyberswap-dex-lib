package uni

import (
	"math/big"
)

var Bone = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	tokenWeightIn uint,
	tokenWeightOut uint,
	swapFee *big.Int,
) (res *big.Int, err error) {
	var amountInWithFee = new(big.Int).Mul(amountIn, new(big.Int).Sub(Bone, swapFee))
	if tokenWeightIn == tokenWeightOut {
		res = new(big.Int).Div(new(big.Int).Mul(reserveOut, amountInWithFee), new(big.Int).Add(new(big.Int).Mul(reserveIn, Bone), amountInWithFee))
		return res, nil
	}
	var baseN = new(big.Int).Add(new(big.Int).Mul(reserveIn, Bone), amountInWithFee)
	res, precision, err := Power(baseN, new(big.Int).Mul(reserveIn, Bone), tokenWeightIn, tokenWeightOut)
	if err != nil {
		return nil, err
	}
	var temp1 = new(big.Int).Mul(reserveOut, res)
	var temp2 = new(big.Int).Lsh(reserveOut, precision)
	res = new(big.Int).Div(new(big.Int).Sub(temp1, temp2), res)
	return res, nil
}
