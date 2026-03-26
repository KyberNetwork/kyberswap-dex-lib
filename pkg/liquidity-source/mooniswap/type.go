package mooniswap

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	Fee         string `json:"fee"`
	SlippageFee string `json:"slpFee"`
	BalAdd0     string `json:"bA0"`
	BalAdd1     string `json:"bA1"`
	BalRem0     string `json:"bR0"`
	BalRem1     string `json:"bR1"`
}

type StaticExtra struct {
	IsNativeToken0 bool `json:"nT0,omitempty"`
	IsNativeToken1 bool `json:"nT1,omitempty"`
}

type PoolMeta struct {
	IsNativeIn  bool `json:"nI,omitempty"`
	IsNativeOut bool `json:"nO,omitempty"`
}

type PoolsListUpdaterMetadata struct {
	TotalPools int `json:"totalPools"`
}

type Config struct {
	DexId          string              `json:"dexId"`
	ChainId        valueobject.ChainID `json:"chainId"`
	FactoryAddress string              `json:"factoryAddress"`
}
