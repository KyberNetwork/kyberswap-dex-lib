package solidlyv2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMetadata struct {
	Dec0     *big.Int
	Dec1     *big.Int
	R0       *big.Int
	R1       *big.Int
	St       bool
	T0       common.Address
	T1       common.Address
	FeeRatio *big.Int
}

type ShadowLegacyMetadata struct {
	Dec0 *big.Int       `abi:"_decimals0"`
	Dec1 *big.Int       `abi:"_decimals1"`
	R0   *big.Int       `abi:"_reserve0"`
	R1   *big.Int       `abi:"_reserve1"`
	St   bool           `abi:"_stable"`
	T0   common.Address `abi:"_token0"`
	T1   common.Address `abi:"_token1"`
}

type MemecoreReserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}
