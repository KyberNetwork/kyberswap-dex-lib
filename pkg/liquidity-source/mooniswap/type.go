package mooniswap

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Extra struct {
	Fee         *uint256.Int `json:"fee"`
	SlippageFee *uint256.Int `json:"slpFee"`
	BalAdd0     *uint256.Int `json:"bA0"`
	BalAdd1     *uint256.Int `json:"bA1"`
	BalRem0     *uint256.Int `json:"bR0"`
	BalRem1     *uint256.Int `json:"bR1"`
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
