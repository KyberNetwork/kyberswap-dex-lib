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
