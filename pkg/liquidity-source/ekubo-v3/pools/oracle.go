package pools

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/quoting"
)

type (
	OraclePoolSwapState = FullRangePoolSwapState
	OraclePoolState     = FullRangePoolState

	OraclePool struct {
		*FullRangePool
		swappedThisBlock bool
	}
)

func (p *OraclePool) CloneState() any {
	cloned := *p
	cloned.FullRangePool = p.FullRangePool.CloneState().(*FullRangePool)
	return &cloned
}

func (p *OraclePool) SetSwapState(state quoting.SwapState) {
	p.FullRangePoolSwapState = state.(*OraclePoolSwapState)
	p.swappedThisBlock = true
}

func (p *OraclePool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.FullRangePool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	if !p.swappedThisBlock {
		quote.Gas += quoting.GasUpdatingOracleSnapshot
	}

	return quote, nil
}

func NewOraclePool(key *FullRangePoolKey, state *OraclePoolState) *OraclePool {
	return &OraclePool{
		FullRangePool:    NewFullRangePool(key, state),
		swappedThisBlock: false,
	}
}
