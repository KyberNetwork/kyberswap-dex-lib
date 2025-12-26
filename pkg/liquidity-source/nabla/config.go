package nabla

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId       string              `json:"dexId"`
	ChainId     valueobject.ChainID `json:"chainId"`
	Portal      string              `json:"portal"`
	Oracle      string              `json:"oracle"`
	Whitelisted string              `json:"whitelisted,omitempty"`
}
