package erc4626

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	ChainId valueobject.ChainID `json:"chainId"`
	DexId   string              `json:"dexId"`
	Vaults  map[string]VaultCfg `json:"vaults"`
}

type VaultCfg struct {
	Gas GasCfg `json:"gas"`
	// MinDepositMethod is an optional no-arg view returning the min deposit assets, e.g. "minAmount".
	MinDepositMethod string `json:"minDepositMethod,omitempty"`
}

type GasCfg struct {
	Deposit uint64 `json:"deposit"`
	Redeem  uint64 `json:"redeem"`
}
