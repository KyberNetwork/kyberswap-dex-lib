package v3

import (
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID           string                `json:"dexID"`
	SubgraphAPI     string                `json:"subgraphAPI"`
	SubgraphHeaders http.Header           `json:"subgraphHeaders"`
	SubgraphTimeout durationjson.Duration `json:"subgraphTimeout"`
	Pairs           map[string]string     `json:"pairs"`
}
