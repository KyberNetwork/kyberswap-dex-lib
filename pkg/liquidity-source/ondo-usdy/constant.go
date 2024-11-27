package ondo_usdy

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
