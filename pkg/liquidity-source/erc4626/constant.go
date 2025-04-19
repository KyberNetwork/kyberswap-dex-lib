package erc4626

import (
	"errors"
)

const (
	DexType = "erc4626"

	erc4626MethodTotalSupply = "totalSupply"
	erc4626MethodTotalAssets = "totalAssets"
	erc4626MethodMaxDeposit  = "maxDeposit"
	erc4626MethodMaxRedeem   = "maxRedeem"
)

var (
	ErrInvalidToken              = errors.New("invalid token")
	ErrUnsupportedSwap           = errors.New("unsupported swap")
	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")
)
