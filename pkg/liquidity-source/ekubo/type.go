package ekubo

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
)

type (
	PoolData struct {
		CoreAddress string `json:"core_address"`
		Token0      string `json:"token0"`
		Token1      string `json:"token1"`
		Fee         string `json:"fee"`
		TickSpacing uint32 `json:"tick_spacing"`
		Extension   string `json:"extension"`
	}

	GetAllPoolsResult = []PoolData
)

type (
	Extra struct {
		quoting.PoolState
	}

	StaticExtra struct {
		ExtensionType pool.ExtensionType `json:"extensionType"`
		PoolKey       *quoting.PoolKey   `json:"poolKey"`
	}
)
