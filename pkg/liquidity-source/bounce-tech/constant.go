package bouncetech

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "bounce-tech"

	mintGas   = 150000
	redeemGas = 200000
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrMintPaused          = errors.New("mint is paused")
	ErrZeroAmount          = errors.New("zero amount")
	ErrInsufficientBalance = errors.New("insufficient base asset balance")
	ErrZeroExchangeRate    = errors.New("zero exchange rate")
	ErrBelowMinAmount      = errors.New("below min transaction size")

	// 1e12 for scaling USDC (6 dec) → 18 dec
	scaleUp = uint256.NewInt(1e12)
	// 1e18 for ScaledNumber.mul / ScaledNumber.div precision
	precision = uint256.NewInt(1e18)
)
