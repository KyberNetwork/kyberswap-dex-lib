package metavault

import "errors"

var (
	ErrVaultSwapsNotEnabled                = errors.New("vault: swaps not enabled")
	ErrVaultMaxUsdmExceeded                = errors.New("vault: max USDM exceeded") // code: 51
	ErrVaultPoolAmountExceeded             = errors.New("vault: poolAmount exceeded")
	ErrVaultReserveExceedsPool             = errors.New("vault: reserve exceeds pool") // code: 50
	ErrVaultPoolAmountLessThanBufferAmount = errors.New("vault: poolAmount < buffer")
	ErrVaultSubOverflow                    = errors.New("vault: SafeMath: subtraction overflow") // transferIn: balanceOf + sent < tokenBalances
	ErrVaultInvalidAmountIn                = errors.New("vault: invalid amountIn")               // code: 27
	ErrVaultPoolAmountExceedsBalance       = errors.New("vault: poolAmount exceeds balance")     // code: 49
	ErrInsufficientRealBalance             = errors.New("vault: real token balance below amount out")

	ErrVaultPriceFeedInvalidPriceFeed         = errors.New("vaultPriceFeed: invalid price feed")
	ErrVaultPriceFeedInvalidPrice             = errors.New("vaultPriceFeed: invalid price")
	ErrVaultPriceFeedCouldNotFetchPrice       = errors.New("vaultPriceFeed: could not fetch price")
	ErrVaultPriceFeedLatestRoundDataReverted  = errors.New("vaultPriceFeed: price feed latestRoundData reverted")
	ErrVaultPriceFeedChainlinkFeedsNotUpdated = errors.New("chainlink feeds are not being updated")

	ErrInvalidSecondaryPriceFeedVersion = errors.New("invalid secondary price feed version")
)
