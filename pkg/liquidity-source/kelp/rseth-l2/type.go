package rsethl2

import "math/big"

type Extra struct {
	SupportedTokenOracles []string   `json:"sTOs"`
	SupportedTokenRates   []*big.Int `json:"sTRates"`
	RSETHRate             *big.Int   `json:"rsETHRate"`
	Fee                   *big.Int   `json:"fee"`
	NativeEnabled         bool       `json:"nativeEnabled"`
}

type PoolExtra struct {
	TokenInIsNative  bool `json:"tokenInIsNative"`
	TokenOutIsNative bool `json:"tokenOutIsNative"`
}
