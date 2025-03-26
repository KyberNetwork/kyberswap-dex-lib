package shared

import "errors"

var (
	ErrEmptyFactoryConfig = errors.New("factory config is empty")

	ErrTokenNotRegistered = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInvalidAmountOut   = errors.New("invalid amount out")

	ErrVaultIsPaused = errors.New("vault is paused")
	ErrPoolIsPaused  = errors.New("pool is paused")
)
