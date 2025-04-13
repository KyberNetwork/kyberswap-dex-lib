package pool

import (
	"math/big"

	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type OraclePool struct {
	BasePool
}

func NewOraclePool(poolKey quoting2.PoolKey, poolState quoting2.PoolState) OraclePool {
	return OraclePool{
		BasePool: NewBasePool(poolKey, poolState),
	}
}

func (p *OraclePool) Quote(amount *big.Int, isToken1 bool) (*quoting2.Quote, error) {
	quote, err := p.BasePool.Quote(amount, isToken1)
	if err != nil {
		return quote, err
	}

	quote.Gas += quoting2.GasCostOfUpdatingOracleSnapshot

	return quote, nil
}
