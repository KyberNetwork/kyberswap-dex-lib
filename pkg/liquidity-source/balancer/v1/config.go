package balancerv1

import (
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID           string                `json:"dexID"`
	NewPoolLimit    int                   `json:"newPoolLimit"`
	SubgraphAPI     string                `json:"subgraphAPI"`
	SubgraphHeaders http.Header           `json:"subgraphHeaders"`
	SubgraphTimeout durationjson.Duration `json:"subgraphTimeout"`
}
