package utils

import (
	"errors"
	"math"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func NewBig(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 0)
	return res
}

func NewFloat(s string) (res *big.Float) {
	res, _ = new(big.Float).SetString(s)
	return res
}

func CalcGasUsd(gasPrice *big.Float, totalGas int64, gasTokenPrice float64) float64 {
	var retFloat = new(big.Float).Quo(
		new(big.Float).Mul(
			new(big.Float).Mul(gasPrice, new(big.Float).SetFloat64(float64(totalGas))),
			new(big.Float).SetFloat64(gasTokenPrice),
		), constant.BoneFloat)
	var ret, _ = retFloat.Float64()
	return ret
}

func CalcTokenAmountUsd(tokenAmount *big.Int, decimals uint8, tokenPrice float64) float64 {
	var retFloat = new(big.Float).Quo(
		new(big.Float).Mul(
			new(big.Float).SetInt(tokenAmount),
			new(big.Float).SetFloat64(tokenPrice),
		),
		constant.TenPowDecimals(decimals),
	)
	var ret, _ = retFloat.Float64()
	return ret
}

func DivDecimals(amount string, decimals uint8) (big.Float, error) {
	var retAmount big.Float
	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return retAmount, errors.New("invalid amount")
	}
	amountBF := new(big.Float).Quo(
		new(big.Float).SetInt(amountBig),
		constant.TenPowDecimals(decimals),
	)

	return *amountBF, nil
}

const float64EqualityThreshold = 1e-9

func AlmostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}
