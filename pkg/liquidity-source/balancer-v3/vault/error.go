package vault

import "errors"

var (
	ErrTradeAmountTooSmall              = errors.New("trade amount is too small")
	ErrProtocolFeesExceedTotalCollected = errors.New("protocolFees exceed totalCollected")
	ErrVaultIsPaused                    = errors.New("vault is paused")
	ErrPoolIsPaused                     = errors.New("pool is paused")
	ErrDynamicSwapFeeHookFailed         = errors.New("dynamicSwapFeeHook is failed")
	ErrPercentageAboveMax               = errors.New("percentage above max")
)
