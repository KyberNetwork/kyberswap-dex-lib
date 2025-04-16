package pool

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type OraclePool struct {
	BasePool
}

func NewOraclePool(poolKey *quoting.PoolKey, poolState quoting.PoolState) OraclePool {
	return OraclePool{
		BasePool: NewBasePool(poolKey, poolState),
	}
}

func (p *OraclePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.BasePool.Quote(amount, isToken1)
	if err != nil {
		return quote, err
	}

	quote.Gas += quoting.GasCostOfUpdatingOracleSnapshot

	return quote, nil
}
