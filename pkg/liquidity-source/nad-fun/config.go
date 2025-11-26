package nadfun

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID             valueobject.ChainID `json:"chainID"`
	DexID               string              `json:"dexID"`
	BondingCurveAddress string              `json:"bondingCurveAddress"`
	RouterAddress       string              `json:"routerAddress"`
	NewPoolLimit        int                 `json:"newPoolLimit"`
}
