package dodo

const (
	TypeV1Pool = "CLASSICAL"
	//TypeVendingMachinePool = "DVM"
	//TypeStablePool         = "DSP"
	//TypePrivatePool        = "DPP"

	rStatusOne      = 0
	rStatusAboveOne = 1
	rStatusBelowOne = 2
)

var (
	DefaultGas = Gas{
		SellBaseV1: 170000,
		BuyBaseV1:  224000,
		SellBaseV2: 128000,
		BuyBaseV2:  116000,
	}
)
