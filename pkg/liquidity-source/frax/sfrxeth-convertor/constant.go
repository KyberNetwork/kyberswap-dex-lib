package sfrxeth_convertor

import (
	"errors"
)

const (
	DexType = "sfrxeth-convertor"
)

var (
	defaultGas = Gas{
		Deposit: 70000,
		Redeem:  50000,
	}
)

var (
	ErrInvalidSwap = errors.New("invalid swap")
	ErrZeroAssets  = errors.New("zero assets")
	ErrZeroDeposit = errors.New("zero deposit")
)
