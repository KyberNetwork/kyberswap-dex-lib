package nadswap

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID             valueobject.ChainID `json:"chainID"`
	DexID               string              `json:"dexID"`
	FactoryAddress      string              `json:"factoryAddress"`
	FeeCollectorAddress string              `json:"feeCollectorAddress"`
	NewPoolLimit        int                 `json:"newPoolLimit"`
}
