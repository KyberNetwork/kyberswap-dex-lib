package nabla

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId           string              `json:"dexId"`
	ChainId         valueobject.ChainID `json:"chainId"`
	Portal          string              `json:"portal"`
	Oracle          string              `json:"oracle"`
	PythAdapterV2   string              `json:"pythAdapterV2"`
	SkipPriceUpdate bool                `json:"skipPriceUpdate"`

	PriceAPI     string                `json:"priceAPI"`
	PriceTimeout durationjson.Duration `json:"priceTimeout"`
}
