package brownfiv3

import (
	"github.com/KyberNetwork/kutils"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID          string              `json:"dexID"`
	ChainID        valueobject.ChainID `json:"chainID"`
	FactoryAddress string              `json:"factoryAddress"`
	NewPoolLimit   int                 `json:"newPoolLimit"`
	Pyth           struct {
		kutils.HttpCfg
		Urls    []string `json:"urls"`
	} `json:"pyth"`
	Multicall3 string `json:"multicall3"`
}
