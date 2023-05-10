package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

func AmountWithoutDecimals(amount *big.Int, decimals uint8) *big.Float {
	result := new(big.Float).SetInt(amount)

	return result.Quo(result, constant.TenPowDecimals(decimals))
}
