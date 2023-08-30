package poolfactory

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

var (
	DefaultGasAlgebra = map[valueobject.Exchange]int64{
		valueobject.ExchangeQuickSwapV3: 280000,
		valueobject.ExchangeSynthSwapV3: 280000,
		valueobject.ExchangeSwapBasedV3: 280000,
	}
)
