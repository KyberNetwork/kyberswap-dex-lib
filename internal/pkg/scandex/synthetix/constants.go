package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type PoolStateVersion uint

const (
	PoolStateVersionNormal PoolStateVersion = 1
	PoolStateVersionAtomic PoolStateVersion = 2
)

var PoolStateVersionByChainID = map[valueobject.ChainID]PoolStateVersion{
	valueobject.ChainIDEthereum: PoolStateVersionAtomic,
	valueobject.ChainIDOptimism: PoolStateVersionNormal,
}

var DefaultPoolStateVersion = PoolStateVersionAtomic

var (
	DefaultChainlinkNumRounds = big.NewInt(5)
)
