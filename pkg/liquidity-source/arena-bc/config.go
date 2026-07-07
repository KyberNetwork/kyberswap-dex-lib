package arenabc

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId        string              `json:"dexId"`
	ChainId      valueobject.ChainID `json:"chainId"`
	TokenManager string              `json:"tokenManager"`
	NewPoolLimit int                 `json:"newPoolLimit"`
}
