package cpmm

import vo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID          string     `json:"dexID"`
	ChainID        vo.ChainID `json:"chainID"`
	VaultAddress   string     `json:"vaultAddress"`
	FactoryAddress string     `json:"factoryAddress"`
	NewPoolLimit   int        `json:"newPoolLimit"`
}
