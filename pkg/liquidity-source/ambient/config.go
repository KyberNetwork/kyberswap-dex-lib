package ambient

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type Config struct {
	DexID                  string                `json:"dexID"`
	SubgraphURL            string                `json:"subgraphUrl"`
	SubgraphRequestTimeout durationjson.Duration `json:"subgraphRequestTimeout"`
}
