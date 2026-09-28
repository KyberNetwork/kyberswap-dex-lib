package prmfun

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexId          string              `json:"dexId"`
	ChainId        valueobject.ChainID `json:"chainId"`
	RouterAddress  string              `json:"routerAddress"`
	FactoryAddress string              `json:"factoryAddress"`
	NewPoolLimit   int                 `json:"newPoolLimit"`
}
