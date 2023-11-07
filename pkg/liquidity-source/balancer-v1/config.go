package balancerv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID                  string                `json:"dexID"`
	NewPoolLimit           int                   `json:"newPoolLimit"`
	SubgraphURL            string                `json:"subgraphUrl"`
	SubgraphRequestTimeout durationjson.Duration `json:"subgraphRequestTimeout"`
}
