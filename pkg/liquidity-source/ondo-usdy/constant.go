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
		Wrap:   550000000,
		Unwrap: 550000000,
	}
)
