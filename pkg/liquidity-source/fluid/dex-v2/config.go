package dexv2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID       string
	ChainID     valueobject.ChainID `json:"chainID"`
	Dex         string              `json:"dex"`
	Resolver    string              `json:"resolver"`
	Liquidity   string              `json:"liquidity"`
	SubgraphAPI string              `json:"subgraphAPI"`
}
