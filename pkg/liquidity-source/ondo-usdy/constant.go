package ondo_usdy

const (
	DexType = "ondo-usdy"

	defaultReserves = "1000000000000000000000000"
)

const (
	mUSDMethodPaused                   = "paused"
	rwaDynamicOracleMethodGetPriceData = "getPriceData"
)

var (
	defaultGas = Gas{
		Wrap:   100000,
		Unwrap: 100000,
	}
)
