package liquiditybookv20

import (
	"net/http"
)

type Config struct {
	DexID              string      `json:"dexID"`
	FactoryAddress     string      `json:"factoryAddress"`
	RouterAddress      string      `json:"routerAddress"`
	NewPoolLimit       int         `json:"newPoolLimit"`
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
}
