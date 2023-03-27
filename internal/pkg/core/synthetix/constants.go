package synthetix

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type PoolStateVersion uint

const (
	PoolStateVersionNormal PoolStateVersion = 1
	PoolStateVersionAtomic PoolStateVersion = 2
)

var (
	DefaultGas = Gas{ExchangeAtomically: 600000, Exchange: 130000}

	PoolStateVersionByChainID = map[valueobject.ChainID]PoolStateVersion{
		valueobject.ChainIDEthereum: PoolStateVersionAtomic,
		valueobject.ChainIDOptimism: PoolStateVersionNormal,
	}

	DefaultPoolStateVersion = PoolStateVersionAtomic
)
