package v2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID     valueobject.ChainID `json:"chainID"`
	DexID       string              `json:"dexID"`
	Comptroller string              `json:"comptroller"`
}
