package classical

const (
	PoolType = "dodo-classical"

	rStatusOne      = 0
	rStatusAboveOne = 1
	rStatusBelowOne = 2
)

var (
	DefaultGas = Gas{
		SellBase:  170000,
		SellQuote: 224000,
		BuyBase:   224000,
	}
)
