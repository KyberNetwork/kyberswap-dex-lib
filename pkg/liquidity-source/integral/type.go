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

	Price         *uint256.Int // Token X -> Y
	InvertedPrice *uint256.Int // Token Y -> X
	SwapFee       *uint256.Int

	Token0LimitMin *uint256.Int
	Token0LimitMax *uint256.Int

	Token1LimitMin *uint256.Int
	Token1LimitMax *uint256.Int
}

type SwapInfo struct {
	RelayerAddress string   `json:"relayerAddress"`
	NewReserve0    *big.Int `json:"-"`
	NewReserve1    *big.Int `json:"-"`
}
