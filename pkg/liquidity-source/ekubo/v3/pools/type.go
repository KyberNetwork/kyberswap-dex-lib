package pools

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/holiman/uint256"
)

type (
	PoolState = any
	Pool      interface {
		GetKey() IPoolKey
		GetState() PoolState

		// Only clones fields updated by SetSwapState
		CloneState() any
		SetSwapState(quoting.SwapState)
		ApplyEvent(event Event, data []byte, blockTimestamp uint64) error
		NewBlock()

		Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error)
		CalcBalances() ([]uint256.Int, error)
	}
)
