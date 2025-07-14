package poolparty

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID         string              `json:"dexID"`
	ChainID       valueobject.ChainID `json:"chainID"`
	Oracle        string              `json:"oracle"`
	Exchange      string              `json:"exchange"`
	BoostPriceBps int                 `json:"boostPriceBps"`
}
