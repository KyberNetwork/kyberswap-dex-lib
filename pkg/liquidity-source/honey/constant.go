package honey

import (
	"errors"
)

const (
	DexType         = "honey"
	defaultReserves = "100000000000000000000000"

	defaultGas = 500000
)

var (
	honeyToken = "0xfcbd14dc51f0a4d49d5e53c2e0950e0bc26d0dce"

	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrMaxRedeemAmountExceeded = errors.New("MAX_REDEEM_AMOUNT_EXCEEDED")
	ErrBasketMode              = errors.New("basket mode")
)
