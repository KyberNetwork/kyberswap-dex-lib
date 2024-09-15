package integral

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

type (
	PoolSimulator struct {
		pool.Pool
		IntegralPair
		TokenLimitMin map[string]*uint256.Int
		gas           Gas
	}

	Gas struct {
		Swap int64
	}
)

type IntegralPair struct {
	IsEnabled bool

	X_Decimals uint64
	Y_Decimals uint64

	SpotPrice    *uint256.Int
	AveragePrice *uint256.Int

	SwapFee *uint256.Int

	Token0LimitMin *uint256.Int
	Token1LimitMin *uint256.Int
}

type PriceInfo struct {
	PriceAccumulator *big.Int
	PriceTimestamp   *big.Int
}

type SwapInfo struct {
	newReserve0 *big.Int
	newReserve1 *big.Int
}
