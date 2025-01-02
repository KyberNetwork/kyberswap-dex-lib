package vault

import "errors"

var (
	ErrAmountInTooSmall                 = errors.New("amount in is too small")
	ErrAmountOutTooSmall                = errors.New("amount out is too small")
	ErrProtocolFeesExceedTotalCollected = errors.New("protocolFees exceed totalCollected")
	ErrVaultIsPaused                    = errors.New("vault is paused")
	ErrPoolIsPaused                     = errors.New("pool is paused")
	ErrDynamicSwapFeeHookFailed         = errors.New("dynamicSwapFeeHook is failed")
	ErrPercentageAboveMax               = errors.New("percentage above max")
	ErrSwapLimit                        = errors.New("swap limit error")
	ErrHookAdjustedSwapLimit            = errors.New("hook adjusted swap limit error")
	ErrBeforeSwapHookFailed             = errors.New("beforeSwapHook is failed")
	ErrAfterSwapHookFailed              = errors.New("afterSwapHook is failed")
)
