package fxdx

import "errors"

var (
	ErrVaultSwapsNotEnabled                = errors.New("vault: swaps not enabled")
	ErrVaultMaxUsdfExceeded                = errors.New("vault: max USDF exceeded") // code: 51
	ErrVaultPoolAmountExceeded             = errors.New("vault: poolAmount exceeded")
	ErrVaultReserveExceedsPool             = errors.New("vault: reserve exceeds pool") // code: 50
	ErrVaultPoolAmountLessThanBufferAmount = errors.New("vault: poolAmount < buffer")

	ErrVaultPriceFeedInvalidPriceFeed         = errors.New("vaultPriceFeed: invalid price feed")
	ErrVaultPriceFeedInvalidPrice             = errors.New("vaultPriceFeed: invalid price")
	ErrVaultPriceFeedCouldNotFetchPrice       = errors.New("vaultPriceFeed: could not fetch price")
	ErrVaultPriceFeedChainlinkFeedsNotUpdated = errors.New("chainlink feeds are not being updated")

	ErrInvalidSecondaryPriceFeedVersion = errors.New("invalid secondary price feed version")

	ErrFeeUtilsV2IsNotInitialized = errors.New("feeUtilsV2: is not initialized")
)
