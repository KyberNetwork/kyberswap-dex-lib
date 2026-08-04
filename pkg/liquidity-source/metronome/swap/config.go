package metronomeswap

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID        string              `json:"dexId"`
	ChainID      valueobject.ChainID `json:"chainID"`
	PoolRegistry string              `json:"poolRegistry"`
}
