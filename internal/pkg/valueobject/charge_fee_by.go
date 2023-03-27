package valueobject

type ChargeFeeBy string

const (
	ChargeFeeByCurrencyIn  = "currency_in"
	ChargeFeeByCurrencyOut = "currency_out"
)

var ChargeFeeByValues = []string{
	ChargeFeeByCurrencyIn,
	ChargeFeeByCurrencyOut,
}
