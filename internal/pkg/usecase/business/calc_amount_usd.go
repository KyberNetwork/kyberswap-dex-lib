package business

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

// CalcAmountUSD returns amount in from usd amount
// amount = (amountUSD / price) * 10^decimals
func CalcAmountFromUSD(amountUSD float64, decimals uint8, priceUSD float64) *big.Int {
	amountUSDBI := new(big.Float).SetFloat64(amountUSD)
	priceUSDBI := new(big.Float).SetFloat64(priceUSD)

	amount := amountUSDBI.Mul(amountUSDBI, constant.TenPowDecimals(decimals))
	result, _ := amount.Quo(amount, priceUSDBI).Int(nil)

	return result
}
