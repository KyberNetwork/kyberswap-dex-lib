package pools

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type OraclePoolSwapState struct {
	*FullRangePoolSwapState
	SwappedThisBlock bool `json:"swappedThisBlock"`
}

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

func (p *OraclePool) SetSwapState(state any) {
	oracleState := state.(*OraclePoolSwapState)

	p.FullRangePoolSwapState = oracleState.FullRangePoolSwapState
	p.swappedThisBlock = oracleState.SwappedThisBlock
}

func (p *OraclePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.FullRangePool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	if !p.swappedThisBlock {
		quote.Gas += quoting.GasCostOfUpdatingOracleSnapshot
	}

	fullRangePoolSwapState := quote.SwapInfo.SwapStateAfter.(FullRangePoolSwapState)

	quote.SwapInfo.SwapStateAfter = OraclePoolSwapState{
		FullRangePoolSwapState: &fullRangePoolSwapState,
		SwappedThisBlock:       true,
	}

	return quote, nil
}
