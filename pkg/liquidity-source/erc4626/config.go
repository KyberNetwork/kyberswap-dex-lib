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
	// RedeemsPausedMethod is an optional no-arg view returning bool, true while redeems revert, e.g. "paused".
	// It is read for every redeem-enabled vault, whether or not deposits are enabled.
	RedeemsPausedMethod string `json:"redeemsPausedMethod,omitempty"`
	// MaxAge (seconds, 0 = off) makes the simulator reject deposits and redeems once the pool is older than
	// this, so a failed pause/cap read (nothing saved) stops quoting. Needs a state refresh (heartbeat) interval below it.
	MaxAge int64 `json:"maxAge,omitempty"`
	// IgnoreZeroMaxDeposit marks a vault that reports maxDeposit 0 while it accepts deposits, so a 0 falls back to
	// totalAssets instead of closing deposits.
	IgnoreZeroMaxDeposit bool `json:"ignoreZeroMaxDeposit,omitempty"`
}

type GasCfg struct {
	Deposit uint64 `json:"deposit"`
	Redeem  uint64 `json:"redeem"`
}
