package capricornpamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	Pools   []string            `json:"pools"`
}
