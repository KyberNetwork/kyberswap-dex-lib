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
		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

type IntegralPair struct {
	DecimalsConverter *big.Int
	SwapFee           *uint256.Int
	AveragePrice      *uint256.Int
}

type PriceInfo struct {
	PriceAccumulator *big.Int
	PriceTimestamp   *big.Int
}

type SwapInfo struct {
	newReserveIn  *big.Int
	newReserveOut *big.Int
}
