package dexLite

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID          string              `json:"dexID"`
	ChainID        valueobject.ChainID `json:"chainID"`
	DexLiteAddress string              `json:"dexLiteAddress"`
}
