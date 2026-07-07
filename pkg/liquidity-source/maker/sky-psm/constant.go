package skypsm

import (
	"errors"
)

const (
	DexType = "sky-psm"

	defaultReserves = "100000000000000000000"

	psm3MethodRateProvider           = "rateProvider"
	psm3MethodPocket                 = "pocket"
	ssrOracleMethodGetConversionRate = "getConversionRate"
)

var (
	defaultGas = Gas{
		SwapExactIn: 70000,
	}

	ErrInvalidToken        = errors.New("invalid token")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
