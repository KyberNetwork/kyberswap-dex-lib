package lunarbase

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID          valueobject.ChainID `json:"chainID"`
	DexID            string              `json:"dexID"`
	CoreAddress      string              `json:"coreAddress"`
	PeripheryAddress string              `json:"peripheryAddress"`
	Permit2Address   string              `json:"permit2Address"`
}
