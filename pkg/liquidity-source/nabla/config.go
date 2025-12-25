package nabla

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexId           string `json:"dexId"`
	Portal          string `json:"portal"`
	Oracle          string `json:"oracle"`
	PythAdapterV2   string `json:"pythAdapterV2"`
	SkipPriceUpdate bool   `json:"skipPriceUpdate"`

	PriceAPI     string                `json:"priceAPI"`
	PriceTimeout durationjson.Duration `json:"priceTimeout"`
}
