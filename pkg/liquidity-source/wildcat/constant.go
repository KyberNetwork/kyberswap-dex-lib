package wildcat

import "errors"

const (
	DexType    = "wildcat"
	defaultGas = 10000
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrTokenNotExist         = errors.New("token does not exist in pool assets")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
