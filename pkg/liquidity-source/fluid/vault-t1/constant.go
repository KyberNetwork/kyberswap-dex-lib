package vaultT1

import (
	"errors"
)

const (
	DexType = "fluid-vault-t1"
)

const ( // VaultLiquidationResolver methods
	VLRMethodGetAllSwapPaths    = "getAllSwapPaths"
	VLRMethodGetSwapForProtocol = "getSwapForProtocol"
)

const (
	String1e27 = "1000000000000000000000000000"
)

var (
	ErrInvalidAmountIn     = errors.New("invalid amountIn: must be greater than zero")
	ErrInsufficientReserve = errors.New("insufficient reserve: tokenOut amount exceeds reserve")
	ErrTokenNotFound       = errors.New("token not found in the pool")
)

var (
	defaultGas = Gas{Liquidate: 1250000}
)
