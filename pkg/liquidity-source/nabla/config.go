package nabla

import (
	"github.com/KyberNetwork/kutils"
)

type Config struct {
	DexId         string `json:"dexId"`
	Portal        string `json:"portal"`
	Oracle        string `json:"oracle"`
	PythAdapterV2 string `json:"pythAdapterV2"`
	Pyth          struct {
		kutils.HttpCfg
		URL string `json:"url"`
	} `json:"pyth"`
}
