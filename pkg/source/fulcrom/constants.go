package fulcrom

import "math/big"

const DexTypeFulcrom = "fulcrom"

var (
	DefaultGas         = Gas{Swap: 165000}
	BasisPointsDivisor = big.NewInt(10000)
	PricePrecision     = new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	USDGDecimals       = big.NewInt(18)
	OneUSD             = PricePrecision
)
