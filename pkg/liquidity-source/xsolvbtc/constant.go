package xsolvbtc

import (
	"errors"
	"math/big"
)

const (
	DexType = "xsolvbtc"
)

var (
	BasisPoint = big.NewInt(10000)
	defaultGas = Gas{
		Deposit:  100000,
		Withdraw: 134000,
	}
	// unlimited reserve
	reserves = "100000000000000000000000000"
)

var (
	ErrNavNotSet         = errors.New("nav not set")
	ErrDepositNotAllowed = errors.New("deposit not allowed")
	ErrXSolvBTCAmount    = errors.New("xSolvBTC amount error")
	ErrSolvBTCAmount     = errors.New("solvBTC amount error") // solvBTC amount is less than factor
)
