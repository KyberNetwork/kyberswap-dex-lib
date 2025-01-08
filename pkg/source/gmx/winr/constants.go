package winr

import "math/big"

const DexTypeWinr = "winr"

var (
	PricePrecision = new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	USDWDecimals   = big.NewInt(18)
	OneUSD         = PricePrecision
)
