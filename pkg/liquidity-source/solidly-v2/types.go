package solidlyv2

import (
	"math/big"

	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
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

type MemecoreReserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type PoolStaticExtra struct {
	velodromev2.PoolStaticExtra
	IsMemecoreDEX bool `json:"isMemecoreDEX"`
}
