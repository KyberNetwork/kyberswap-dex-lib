package dexv2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID       string
	ChainID     valueobject.ChainID `json:"chainID"`
	Resolver    string              `json:"resolver"`
	SubgraphAPI string              `json:"subgraphAPI"`
}
