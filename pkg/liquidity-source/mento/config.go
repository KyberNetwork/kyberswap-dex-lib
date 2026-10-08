package mento

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID          string              `json:"dexID"`
	ChainID        valueobject.ChainID `json:"chainID"`
	FactoryAddress string              `json:"factoryAddress"` // FPMMFactory
	NewPoolLimit   int                 `json:"newPoolLimit"`
}
