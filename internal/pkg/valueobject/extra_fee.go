package valueobject

import (
	"math/big"
)

var (
	// BasisPoint is one hundredth of 1 percentage point
	// https://en.wikipedia.org/wiki/Basis_point
	BasisPoint   = big.NewInt(10000)
	ZeroExtraFee = ExtraFee{
		FeeAmount: big.NewInt(0),
	}
)

// ExtraFee is a fee customized by client
type ExtraFee struct {
	FeeAmount   *big.Int    `json:"feeAmount"`
	ChargeFeeBy ChargeFeeBy `json:"chargeFeeBy"`
	IsInBps     bool        `json:"isInBps"`
	FeeReceiver string      `json:"feeReceiver"`
}

func (f ExtraFee) IsChargeFeeByCurrencyIn() bool {
	return f.ChargeFeeBy == ChargeFeeByCurrencyIn
}

func (f ExtraFee) IsChargeFeeByCurrencyOut() bool {
	return f.ChargeFeeBy == ChargeFeeByCurrencyOut
}

// CalcActualFeeAmount returns actual fee amount
// - if IsInBps == true: actualFeeAmount = FeeAmount
// - otherwise: actualFeeAmount = amount * FeeAmount / BasisPoint
func (f ExtraFee) CalcActualFeeAmount(amount *big.Int) *big.Int {
	if !f.IsInBps {
		return f.FeeAmount
	}

	return new(big.Int).Div(
		new(big.Int).Mul(amount, f.FeeAmount),
		BasisPoint,
	)
}
