package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// CalcAmountInAfterFee returns amount of token to be swapped after extra fee
// - if ChargeFeeBy is different from currency_in: amountInAfterFee = amountIn
// - otherwise: amountInAfterFee = amountIn - actualFeeAmount
func CalcAmountInAfterFee(amountIn *big.Int, extraFee valueobject.ExtraFee) *big.Int {
	if extraFee.ChargeFeeBy != valueobject.ChargeFeeByCurrencyIn {
		return amountIn
	}

	return new(big.Int).Sub(amountIn, extraFee.CalcActualFeeAmount(amountIn))
}

// CalcAmountOutAfterFee returns amount of token to be received after extra fee
// - if ChargeFeeBy is different from currency_out: amountOutAfterFee = amountIn
// - otherwise: amountOutAfterFee = amountOut - actualFeeAmount
func CalcAmountOutAfterFee(amountOut *big.Int, extraFee valueobject.ExtraFee) *big.Int {
	if extraFee.ChargeFeeBy != valueobject.ChargeFeeByCurrencyOut {
		return amountOut
	}

	return new(big.Int).Sub(amountOut, extraFee.CalcActualFeeAmount(amountOut))
}
