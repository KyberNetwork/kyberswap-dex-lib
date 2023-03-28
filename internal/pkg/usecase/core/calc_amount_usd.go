package core

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// CalcAmountUSD returns amount in USD
// amountUSD = (amount / 10^decimals) * priceInUSD
func CalcAmountUSD(amount *big.Int, decimals uint8, priceUSD float64) *big.Float {
	return new(big.Float).Mul(
		new(big.Float).Quo(
			new(big.Float).SetInt(amount),
			constant.TenPowDecimals(decimals),
		),
		new(big.Float).SetFloat64(priceUSD),
	)
}
