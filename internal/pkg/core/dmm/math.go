package dmm

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

func GetAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	vReserveIn *big.Int,
	vReserveOut *big.Int,
	feeInPrecision *big.Int,
) (*big.Int, error) {
	if amountIn.Cmp(constant.Zero) <= 0 {
		return nil, errors.New("DMM: INSUFFICIENT_INPUT_AMOUNT")
	}
	if reserveIn.Cmp(constant.Zero) <= 0 || reserveOut.Cmp(constant.Zero) <= 0 {
		return nil, errors.New("DMM: INSUFFICIENT_LIQUIDITY")
	}
	var amountInWithFee = new(big.Int).Div(
		new(big.Int).Mul(amountIn, new(big.Int).Sub(constant.BONE, feeInPrecision)),
		constant.BONE,
	)
	var numerator = new(big.Int).Mul(amountInWithFee, vReserveOut)
	var denominator = new(big.Int).Add(vReserveIn, amountInWithFee)
	var amountOut = new(big.Int).Div(numerator, denominator)
	if amountOut.Cmp(reserveOut) >= 0 {
		return nil, errors.New("DMM: INSUFFICIENT_LIQUIDITY")
	}
	return amountOut, nil
}
