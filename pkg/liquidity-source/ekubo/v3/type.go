package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	PoolState = any

	Extra = PoolState

	StaticExtra struct {
		Core             common.Address    `json:"core"`
		ExtensionType    ExtensionType     `json:"extensionType"`
		PoolKey          *pools.AnyPoolKey `json:"poolKey"`
		MevCaptureRouter common.Address    `json:"mevCaptureRouter"`
	}

	Meta struct {
		MevCaptureRouter common.Address   `json:"router"`
		Core             common.Address   `json:"core"`
		PoolKey          pools.AbiPoolKey `json:"poolKey"`
	}

	Pool interface {
		GetKey() pools.IPoolKey
		GetState() PoolState

		CloneState() any
		SetSwapState(quoting.SwapState)
		ApplyEvent(event pools.Event, data []byte, blockTimestamp uint64) error
		NewBlock()

		Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error)
		CalcBalances() ([]uint256.Int, error)
	}

	PoolWithBlockNumber struct {
		Pool
		blockNumber uint64
	}
)
