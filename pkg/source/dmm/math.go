package dmm

import (
	"errors"
	"math/big"
)

var (
	ErrInsufficientInputAmount = errors.New("DMM: INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("DMM: INSUFFICIENT_LIQUIDITY")
)

func GetAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	vReserveIn *big.Int,
	vReserveOut *big.Int,
	feeInPrecision *big.Int,
) (*big.Int, error) {
	if amountIn.Cmp(zeroBI) <= 0 {
		return nil, ErrInsufficientInputAmount
	}
	if reserveIn.Cmp(zeroBI) <= 0 || reserveOut.Cmp(zeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	var amountInWithFee = new(big.Int).Div(
		new(big.Int).Mul(amountIn, new(big.Int).Sub(bONE, feeInPrecision)),
		bONE,
	)
	var numerator = new(big.Int).Mul(amountInWithFee, vReserveOut)
	var denominator = new(big.Int).Add(vReserveIn, amountInWithFee)
	var amountOut = new(big.Int).Div(numerator, denominator)
	if amountOut.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}
	return amountOut, nil
}

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}
