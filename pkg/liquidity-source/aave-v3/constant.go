package aavev3

const (
	DexType = "aave-v3"

	supplyGas   int64 = 150000
	withdrawGas int64 = 150000

	defaultReserve = 10000000000

	poolMethodGetReserveAToken      = "getReserveAToken"
	poolMethodGetReserveAddressById = "getReserveAddressById"
	poolMethodGetReservesList       = "getReservesList"
	poolMethodGetReservesCount      = "getReservesCount"
	poolMethodGetConfiguration      = "getConfiguration"

	aTokenMethodScaledTotalSupply = "scaledTotalSupply"
)
