package quickperps

import "errors"

var (
	ErrVaultSwapsNotEnabled                = errors.New("vault: swaps not enabled")
	ErrVaultMaxUsdqExceeded                = errors.New("vault: max USDQ exceeded") // code: 51
	ErrVaultPoolAmountExceeded             = errors.New("vault: poolAmount exceeded")
	ErrVaultReserveExceedsPool             = errors.New("vault: reserve exceeds pool") // code: 50
	ErrVaultPoolAmountLessThanBufferAmount = errors.New("vault: poolAmount < buffer")

	ErrVaultPriceFeedInvalidPriceFeed = errors.New("vaultPriceFeed: invalid price feed")
	ErrVaultPriceFeedInvalidPrice     = errors.New("vaultPriceFeed: invalid price")
	ErrVaultPriceFeedExpired          = errors.New("vaultPriceFeed: expired")

	ErrInvalidSecondaryPriceFeedVersion = errors.New("invalid secondary price feed version")
)
