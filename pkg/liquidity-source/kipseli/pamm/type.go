package pamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

type Extra = kipseli.Extra

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	BlockNumber      uint64                       `json:"blockNumber"`
	RouterAddress    string                       `json:"routerAddress"`
	SO               map[string]map[string]string `json:"so,omitempty"`
	LastUpdatedBlock uint64                       `json:"lub,omitempty"`
}
