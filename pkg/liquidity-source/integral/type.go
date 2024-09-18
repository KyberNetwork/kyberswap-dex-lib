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
	RelayerAddress string

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
	RelayerAddress string   `json:"relayerAddress"`
	NewReserve0    *big.Int `json:"-"`
	NewReserve1    *big.Int `json:"-"`
}
