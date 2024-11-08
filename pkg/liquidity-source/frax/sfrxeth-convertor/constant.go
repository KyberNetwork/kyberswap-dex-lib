package sfrxeth_convertor

const (
	DexType = "sfrxeth-convertor"

	defaultReserves = "1000000000000000000000000"
)

const (
	sfrxETHMethodTotalAssets = "totalAssets"
	sfrxETHMethodTotalSupply = "totalSupply"
)

const (
	Deposit = iota
	Redeem
	InvalidSwap
)

var (
	defaultGas = Gas{
		Deposit: 250000,
		Redeem:  250000,
	}
)
