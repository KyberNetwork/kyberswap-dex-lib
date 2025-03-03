package utils

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

const float64EqualityThreshold = 1e-9

func Float64AlmostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func NewBig(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 0)
	return res
}

func CeilBigFloat(bf *big.Float) *big.Int {
	res, accuracy := bf.Int(nil)
	if accuracy == big.Below {
		res.Add(res, bignumber.One)
	}
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
