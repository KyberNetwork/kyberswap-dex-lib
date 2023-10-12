package gmxglp

import "errors"

var (
	ErrVaultSwapsNotEnabled                = errors.New("vault: swaps not enabled")
	ErrVaultMaxUsdgExceeded                = errors.New("vault: max USDG exceeded") // code: 51
	ErrVaultPoolAmountExceeded             = errors.New("vault: poolAmount exceeded")
	ErrVaultReserveExceedsPool             = errors.New("vault: reserve exceeds pool") // code: 50
	ErrVaultPoolAmountLessThanBufferAmount = errors.New("vault: poolAmount < buffer")
	ErrVaultNegativeTokenAmount            = errors.New("vault: tokenAmount < 0")      // code: 17
	ErrVaultNegativeUsdgAmount             = errors.New("vault: usdgAmount < 0")       // code: 18
	ErrVaultNegativeRedemptionAmount       = errors.New("vault: redemptionAmount < 0") // code: 20
	ErrVaultNegativeAmountOut              = errors.New("vault: amountOut < 0 ")       // ocde: 22

	ErrVaultPriceFeedInvalidPriceFeed         = errors.New("vaultPriceFeed: invalid price feed")
	ErrVaultPriceFeedInvalidPrice             = errors.New("vaultPriceFeed: invalid price")
	ErrVaultPriceFeedCouldNotFetchPrice       = errors.New("vaultPriceFeed: could not fetch price")
	ErrVaultPriceFeedChainlinkFeedsNotUpdated = errors.New("chainlink feeds are not being updated")

	ErrInvalidSecondaryPriceFeedVersion = errors.New("invalid secondary price feed version")

	ErrRewardRouterInvalidAmount    = errors.New("rewardRouter: invalid amount")
	ErrRewardRouterInvalidGlpAmount = errors.New("rewardRouter: invalid glpAmount")
	ErrGlpManagerInvalidAmount      = errors.New("glpManager: invalid _amount")

	ErrSafeMathMulOverflow = errors.New("safeMath: multiplication overflow")
	ErrSafeMathDivZero     = errors.New("safeMath: division by zero")
	ErrSafeMathSubOverflow = errors.New("safeMath: subtraction overflow")
	ErrSafeMathAddOverflow = errors.New("safeMath: addition overflow")

	ErrYearnTokenVaultDepositNotRespected = errors.New("deposit limit is not respected")
	ErrYearnTokenVaultDepositNothing      = errors.New("deposit nothing")
	ErrYearnTokenVaultWithdrawNothing     = errors.New("withdraw nothing")
)
