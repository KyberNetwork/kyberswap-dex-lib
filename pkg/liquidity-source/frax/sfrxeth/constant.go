package sfrxeth

const (
	DexType = "sfrxeth"
)

const (
	minterMethodSubmitPaused = "submitPaused"

	SfrxETHMethodTotalAssets = "totalAssets"
	SfrxETHMethodTotalSupply = "totalSupply"
)

var (
	defaultGas = Gas{
		SubmitAndDeposit: 90000,
	}
)
