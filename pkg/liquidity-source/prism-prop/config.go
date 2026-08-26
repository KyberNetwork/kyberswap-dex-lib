package prismprop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID         string              `json:"dexId"`
	ChainID       valueobject.ChainID `json:"chainId"`
	RouterAddress string              `json:"routerAddress"`
}
