package core

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// CalcL1FeeUSD return gas in USD that is used to publish tx to L1
// l1GasUSD = (l1GasFee / 10^18) * tokenPriceUSD
func CalcL1FeeUSD(l1GasFee *big.Int, gasTokenPriceUSD float64) *big.Float {
	return new(big.Float).Mul(
		new(big.Float).Quo(
			new(big.Float).SetInt(l1GasFee),
			constant.BoneFloat,
		),
		new(big.Float).SetFloat64(gasTokenPriceUSD),
	)
}
