package shared

import "errors"

var (
	ErrTokenNotRegistered = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInvalidAmountOut   = errors.New("invalid amount out")
	ErrInvalidPoolType    = errors.New("invalid pool type")
	ErrInvalidPoolID      = errors.New("invalid pool id")

	ErrTradeAmountTooSmall              = errors.New("trade amount is too small")
	ErrProtocolFeesExceedTotalCollected = errors.New("protocolFees exceed totalCollected")
	ErrVaultIsPaused                    = errors.New("vault is paused")
	ErrPoolIsPaused                     = errors.New("pool is paused")
)
