package poolfactory

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

var (
	DefaultGasAlgebra = map[valueobject.Exchange]int64{
		valueobject.ExchangeQuickSwapV3: 280000,
		valueobject.ExchangeSynthSwapV3: 280000,
		valueobject.ExchangeSwapBasedV3: 280000,
		valueobject.ExchangeLynex:       280000,
		valueobject.ExchangeCamelotV3:   280000,
		valueobject.ExchangeZyberSwapV3: 280000,
		valueobject.ExchangeThenaFusion: 280000,
	}
)
