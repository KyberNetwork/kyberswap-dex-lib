package utils

import (
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
