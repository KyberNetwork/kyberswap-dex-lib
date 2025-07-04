package pools

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type OraclePoolSwapState = FullRangePoolSwapState
type OraclePoolState = FullRangePoolState

type OraclePool struct {
	*FullRangePool
	swappedThisBlock bool
}

func NewOraclePool(key *PoolKey, state *OraclePoolState) *OraclePool {
	return &OraclePool{
		FullRangePool:    NewFullRangePool(key, state),
		swappedThisBlock: false,
	}
}

func (p *OraclePool) CloneState() any {
	cloned := *p
	cloned.FullRangePool = p.FullRangePool.CloneState().(*FullRangePool)
	return &cloned
}

func (p *OraclePool) SetSwapState(state quoting.SwapState) {
	p.FullRangePoolSwapState = state.(*OraclePoolSwapState)
	p.swappedThisBlock = true
}

func (p *OraclePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.FullRangePool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	if !p.swappedThisBlock {
		quote.Gas += quoting.GasCostOfUpdatingOracleSnapshot
	}

	return quote, nil
}
