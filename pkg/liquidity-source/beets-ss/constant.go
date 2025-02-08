package beets_ss

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"
)

const (
	DexType                    = "beets-ss"
	Beets_Staked_Sonic_Address = "0xe5da20f15420ad15de0fa650600afc998bbe3955"
	defaultReserve             = "100000000000000000000000000"
)

var (
	defaultGas = Gas{Swap: 60000}

	MIN_DEPOSIT = uint256.NewInt(1e16)
	ZERO        = big.NewInt(0)

	methodTotalSupply   = "totalSupply"
	methodTotalAssets   = "totalAssets"
	methodDepositPaused = "depositPaused"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")

	ErrDepositTooSmall = errors.New("deposit too small")
	ErrDepositPaused   = errors.New("deposit paused")
	ErrOverflow        = errors.New("overflow")
)
