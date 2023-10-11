package entity

import "github.com/ethereum/go-ethereum/common"

type ERC20BalanceSlot struct {
	// The token itself.
	Token string `mapstructure:"token" json:"token"`
	// The wallet whose balance is going to be faked.
	Wallet string `mapstructure:"wallet" json:"wallet,omitempty"`
	// True if either balance slot && its overrides or holderWallet is found.
	Found bool `mapstructure:"found" json:"found"`
	// The storage slot where wallet's balance is stored.
	BalanceSlot string `mapstructure:"balanceSlot" json:"balanceSlot,omitempty"`
	// If not empty, use this value as balanceSlot's value instead.
	PreferredValue string `mapstructure:"preferredValue" json:"preferredValue,omitempty"`
	// Extra overrides needed to make the fake balance valid.
	ExtraOverrides map[string]string `mapstructure:"extraOverrides" json:"extraOverrides,omitempty"`
	// The list of attempted strategies.
	StrategiesAttempted []string `mapstructure:"strategiesAttempted" json:"strategiesAttempted,omitempty"`
}

type TokenBalanceSlots = map[common.Address]*ERC20BalanceSlot
