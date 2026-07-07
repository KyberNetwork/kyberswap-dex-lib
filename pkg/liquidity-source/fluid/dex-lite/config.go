package dexLite

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID           string              `json:"dexID"`
	ChainID         valueobject.ChainID `json:"chainID"`
	DexLiteAddress  string              `json:"dexLiteAddress"`
	DeployerAddress common.Address      `json:"deployerAddress"`
}
