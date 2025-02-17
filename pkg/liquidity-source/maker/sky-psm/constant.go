package skypsm

import (
	"errors"
)

const (
	DexType = "sky-psm"

	defaultReserves = "100000000000000000000"

	ssrOracleMethodGetConversionRate = "getConversionRate"
)

var (
	defaultGas = Gas{
		SwapExactIn: 70000,
	}
)

var (
	ErrInvalidToken = errors.New("invalid token")
)
