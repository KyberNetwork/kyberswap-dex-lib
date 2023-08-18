package entity

import "github.com/ethereum/go-ethereum/common"

type ERC20BalanceSlot struct {
	Token          string            `mapstructure:"token" json:"token"`
	Wallet         string            `mapstructure:"wallet" json:"wallet"`
	Found          bool              `mapstructure:"found" json:"found"`
	BalanceSlot    string            `mapstructure:"balanceSlot" json:"balanceSlot,omitempty"`
	PreferredValue string            `mapstructure:"preferredValue" json:"preferredValue,omitempty"`
	ExtraOverrides map[string]string `mapstructure:"extraOverrides" json:"extraOverrides,omitempty"`
}

type TokenBalanceSlots = map[common.Address]*ERC20BalanceSlot
