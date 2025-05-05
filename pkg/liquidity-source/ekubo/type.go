package ekubo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type (
	PoolState = any

	Extra = PoolState

	StaticExtra struct {
		Core          common.Address `json:"core"`
		ExtensionType ExtensionType  `json:"extensionType"`
		PoolKey       *pools.PoolKey `json:"poolKey"`
	}

	Meta struct {
		Core    common.Address   `json:"core"`
		PoolKey pools.AbiPoolKey `json:"poolKey"`
	}

	Pool interface {
		GetKey() *pools.PoolKey
		GetState() PoolState

		SetSwapState(quoting.SwapState)
		ApplyEvent(event pools.Event, data []byte, blockTimestamp uint64) error
		NewBlock()

		Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error)
		CalcBalances() ([]big.Int, error)
	}

	PoolWithBlockNumber struct {
		Pool
		blockNumber uint64
	}
)
