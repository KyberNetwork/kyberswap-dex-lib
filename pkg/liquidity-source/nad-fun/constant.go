package nadfun

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "nad-fun"

	buyGas  = 270000
	sellGas = 200000
)

var (
	ErrPoolLocked            = errors.New("pool is locked")
	ErrPoolGraduated         = errors.New("pool is graduated")
	ErrInvalidToken          = errors.New("invalid token")
	ErrOverflow              = errors.New("overflow")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInvariantViolation    = errors.New("k invariant violated")
	ErrTargetExceeded        = errors.New("target token amount exceeded")

	FeeDenom = uint256.NewInt(1000000)
)
