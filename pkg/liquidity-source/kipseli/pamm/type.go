package pamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

type Extra = kipseli.Extra

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	RouterAddress    string                           `json:"routerAddress"`
	BlockNumber      uint64                           `json:"bn"`
	SO               map[string]kipseli.StateOverride `json:"so,omitempty"`
	LastUpdatedBlock uint64                           `json:"lub,omitempty"`
}
