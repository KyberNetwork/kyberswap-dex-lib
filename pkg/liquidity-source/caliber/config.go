package caliber

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID    string              `json:"dexID"`
	ChainID  valueobject.ChainID `json:"chainID"`
	Contract string              `json:"contract"`
	Pairs    []PairConfig        `json:"pairs"`
}

type PairConfig struct {
	PairID string `json:"pairId"`
	Token0 string `json:"token0"`
	Token1 string `json:"token1"`
}
