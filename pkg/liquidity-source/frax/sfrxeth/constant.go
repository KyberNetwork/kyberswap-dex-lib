package sfrxeth

const (
	DexType = "sfrxeth"

	defaultReserves = "1000000000000000000000000"
)

const (
	minterMethodSubmitPaused = "submitPaused"
	sfrxETHMethodTotalAssets = "totalAssets"
	sfrxETHMethodTotalSupply = "totalSupply"
)

var (
	defaultGas = Gas{
		SubmitAndDeposit: 250000,
	}
)
