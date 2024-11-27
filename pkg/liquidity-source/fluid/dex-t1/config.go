package dexT1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID               string              `json:"dexID"`
	ChainID             valueobject.ChainID `json:"chainID"`
	DexReservesResolver string              `json:"dexReservesResolver"`
}
