package shared

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID             string              `json:"dexID"`
	ChainID           valueobject.ChainID `json:"chainID"`
	SubgraphAPI       string              `json:"subgraphAPI"`
	SubgraphHeaders   http.Header         `json:"subgraphHeaders"`
	NewPoolLimit      int                 `json:"newPoolLimit"`
	DodoV1SellHelper  string              `json:"dodoV1SellHelper"`
	BlacklistFilePath string              `json:"blacklistFilePath"`
}
