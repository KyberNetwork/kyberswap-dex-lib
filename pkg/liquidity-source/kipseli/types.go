package kipseli

import (
	"errors"
	"math/big"
)

// StateOverride is one contract entry from the Titan stream — passes through
// untouched to downstream simulation tools (Tenderly state_objects, eth_call
// overrides). Empty fields are omitted from JSON.
type StateOverride struct {
	Storage map[string]string `json:"storage,omitempty"`
	Balance string            `json:"balance,omitempty"`
	Nonce   string            `json:"nonce,omitempty"`
}

type Extra struct {
	Samples        [][][2]*big.Int          `json:"samples"`
	MaxIn          []*big.Int               `json:"maxIn,omitempty"`
	SO             map[string]StateOverride `json:"so,omitempty"`
	BlockTimestamp uint64                   `json:"bt,omitempty"`
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
