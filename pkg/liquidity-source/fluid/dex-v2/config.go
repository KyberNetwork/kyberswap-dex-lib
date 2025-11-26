package dexv2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID       string
	ChainID     valueobject.ChainID `json:"chainID"`
	SubgraphAPI string              `json:"subgraphAPI"`
}
