package valueobject

import (
	"math/big"

	"github.com/samber/lo"
)

var (
	// BasisPoint is one hundredth of 1 percentage point
	// https://en.wikipedia.org/wiki/Basis_point
	BasisPoint   = big.NewInt(10000)
	ZeroExtraFee = ExtraFee{}
)

// ExtraFee is a fee customized by client
type ExtraFee struct {
	FeeAmount   []*big.Int  `json:"feeAmount"`
	ChargeFeeBy ChargeFeeBy `json:"chargeFeeBy"`
	IsInBps     bool        `json:"isInBps"`
	FeeReceiver []string    `json:"feeReceiver"`
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
	feeSum := lo.Reduce(f.FeeAmount, func(agg *big.Int, item *big.Int, i int) *big.Int {
		return agg.Add(agg, item)
	}, big.NewInt(0))

	if !f.IsInBps {
		return feeSum
	}
	return new(big.Int).Div(
		new(big.Int).Mul(amount, feeSum),
		BasisPoint,
	)
}
