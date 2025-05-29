package honey

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType         = "honey"
	defaultReserves = "100000000000000000000000"

	defaultGas = 500000
)

var (
	honeyToken = "0xfcbd14dc51f0a4d49d5e53c2e0950e0bc26d0dce"

	U10   = uint256.NewInt(10)
	U1e18 = uint256.NewInt(1e18)

	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrMaxRedeemAmountExceeded = errors.New("MAX_REDEEM_AMOUNT_EXCEEDED")
	ErrBasketMode              = errors.New("basket mode")
)
