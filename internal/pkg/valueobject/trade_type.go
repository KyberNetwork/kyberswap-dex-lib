package valueobject

type TradeType string

const (
	TradeTypeExactInput  TradeType = "EXACT_INPUT"
	TradeTypeExactOutput TradeType = "EXACT_OUTPUT"
)

func (t TradeType) IsExactInput() bool {
	return t == TradeTypeExactInput
}

func (t TradeType) IsExactOutput() bool {
	return t == TradeTypeExactOutput
}
