package cloberob

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId              string              `json:"dexId"`
	ChainId            valueobject.ChainID `json:"chainId"`
	SubgraphAPI        string              `json:"subgraphAPI"`
	NewPoolLimit       int                 `json:"newPoolLimit"`
	AllowSubgraphError bool                `json:"allowSubgraphError"`
	BookManager        common.Address      `json:"bookManager"`
	BookViewer         common.Address      `json:"bookViewer"`
}
