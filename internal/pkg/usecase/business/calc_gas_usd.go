package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// CalcGasUSD return gas in USD
// gasUSD = (gasPrice / 10^18) * totalGas * tokenPriceUSD
func CalcGasUSD(gasPrice *big.Float, totalGas int64, gasTokenPriceUSD float64) *big.Float {
	return new(big.Float).Mul(
		new(big.Float).Mul(
			new(big.Float).Quo(
				gasPrice,
				constant.BoneFloat,
			),
			new(big.Float).SetInt64(totalGas),
		),
		new(big.Float).SetFloat64(gasTokenPriceUSD),
	)
}
