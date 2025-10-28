package brownfiv2

import (
	"github.com/KyberNetwork/kutils"
)

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
	Pyth           struct {
		kutils.HttpCfg
		Urls []string `json:"urls"`
	} `json:"pyth"`
}
