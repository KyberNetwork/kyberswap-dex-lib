package parityswapprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID    string              `json:"dexID"`
	ChainID  valueobject.ChainID `json:"chainId"`
	Registry string              `json:"registry"`
}
