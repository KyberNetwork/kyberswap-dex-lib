package clear

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID            string              `json:"dexID"`
	ChainID          valueobject.ChainID `json:"chainId"`
	NewPoolLimit     int                 `json:"newPoolLimit"`
	SwapAddress      string              `json:"swapAddress"`
	FactoryAddresses []string            `json:"factoryAddresses"`
}
