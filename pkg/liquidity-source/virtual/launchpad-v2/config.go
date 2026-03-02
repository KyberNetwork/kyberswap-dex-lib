package launchpadv2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID      valueobject.ChainID  `json:"chainId"`
	DexId        valueobject.Exchange `json:"dexId"`
	Factory      string               `json:"factory"`
	NewPoolLimit int                  `json:"newPoolLimit"`
}
