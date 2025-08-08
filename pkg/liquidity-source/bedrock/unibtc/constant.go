package unibtc

import (
	"errors"
)

const (
	DexType = "bedrock-unibtc"
)

var (
	Vault = "0x4befa2aa9c305238aa3e0b5d17eb20c045269e9d"
)

var (
	defaultGas = Gas{
		Mint: 100000,
	}
	// unlimited reserve
	reserves = "10000000000000000000"
)

var (
	ErrUnsupportedToken = errors.New("paused or not allowed")
	ErrUnsupportedSwap  = errors.New("unsupported swap")
	ErrInsufficientCap  = errors.New("insufficient cap")
)
