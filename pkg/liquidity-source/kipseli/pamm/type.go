package pamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

type Extra = kipseli.Extra

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

type PoolMetaInfo struct {
	RouterAddress  string                           `json:"routerAddress"`
	BlockNumber    uint64                           `json:"bn"`
	BlockTimestamp uint64                           `json:"bt,omitempty"`
	SO             map[string]kipseli.StateOverride `json:"so,omitempty"`
}
