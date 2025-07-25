package valueobject

import (
	encodeValueObject "github.com/KyberNetwork/aggregator-encoding/pkg/constant/valueobject"
)

type ChargeFeeBy = encodeValueObject.ChargeFeeBy

const (
	ChargeFeeByCurrencyIn  = encodeValueObject.ChargeFeeByCurrencyIn
	ChargeFeeByCurrencyOut = encodeValueObject.ChargeFeeByCurrencyOut
)

var ChargeFeeByValues = []ChargeFeeBy{
	ChargeFeeByCurrencyIn,
	ChargeFeeByCurrencyOut,
}
