package virtualfun

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID               valueobject.ChainID `json:"chainID"`
	BondingAddress        string              `json:"bondingAddress"`
	FactoryAddress        string              `json:"factoryAddress"`
	AssetToken            string              `json:"assetToken"`
	IgnoreUntradablePools bool                `json:"ignoreUntradablePools"`
	NewPoolLimit          int                 `json:"newPoolLimit"`
}
