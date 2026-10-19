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
	// DepositsPausedMethod is an optional no-arg view returning bool, true while deposits revert, e.g. "depositsPaused".
	DepositsPausedMethod string `json:"depositsPausedMethod,omitempty"`
	// MaxSupplyMethod is an optional no-arg view returning the share supply cap, e.g. "maxSupply". Deposits that
	// mint shares beyond maxSupply - totalSupply revert, though maxDeposit/maxMint don't show it.
	MaxSupplyMethod string `json:"maxSupplyMethod,omitempty"`
}

type GasCfg struct {
	Deposit uint64 `json:"deposit"`
	Redeem  uint64 `json:"redeem"`
}
