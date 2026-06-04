package maplesyrup

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"

type Config struct {
	DexId  string              `json:"dexId"`
	Vaults map[string]VaultCfg `json:"vaults"`
}

type VaultCfg struct {
	Gas         erc4626.GasCfg `json:"gas"`
	Router      string         `json:"router"`
	PoolManager string         `json:"poolManager"`
}
