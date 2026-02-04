package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

type (
	Extra = pools.PoolState

	StaticExtra struct {
		Core             common.Address   `json:"core"`
		ExtensionType    ExtensionType    `json:"extensionType"`
		PoolKey          pools.AnyPoolKey `json:"poolKey"`
		MevCaptureRouter common.Address   `json:"mevCaptureRouter"`
	}

	Meta struct {
		MevCaptureRouter common.Address   `json:"router"`
		Core             common.Address   `json:"core"`
		PoolKey          pools.AbiPoolKey `json:"poolKey"`
	}

	PoolWithBlockNumber struct {
		pools.Pool
		blockNumber uint64
	}
)
