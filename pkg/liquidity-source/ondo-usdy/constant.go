package ondo_usdy

import (
	"errors"
)

const (
	DexType = "ondo-usdy"

	defaultReserves = "1000000000000000000000000"
)

const (
	rUSDYMethodPaused = "paused"

	// on ethereum
	rUSDYMethodTotalShares = "totalShares"
	// on mantle
	rUSDYWMethodGetTotalShares = "getTotalShares"

	rwaDynamicOracleMethodGetPriceData = "getPriceData"
)

var (
	defaultGas = Gas{
		Wrap:   100000,
		Unwrap: 100000,
	}
)

var (
	ErrPoolPaused     = errors.New("pool is paused")
	ErrUnwrapTooSmall = errors.New("unwrap too small")
)
