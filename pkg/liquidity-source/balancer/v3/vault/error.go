package vault

import "errors"

var (
	ErrAmountInTooSmall                 = errors.New("amount in is too small")
	ErrAmountOutTooSmall                = errors.New("amount out is too small")
	ErrProtocolFeesExceedTotalCollected = errors.New("protocolFees exceed totalCollected")
	ErrDynamicSwapFeeHookFailed         = errors.New("dynamicSwapFeeHook is failed")
	ErrBeforeSwapHookFailed             = errors.New("beforeSwapHook is failed")
	ErrAfterSwapHookFailed              = errors.New("afterSwapHook is failed")
)
