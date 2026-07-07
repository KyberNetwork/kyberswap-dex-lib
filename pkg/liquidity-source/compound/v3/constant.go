package v3

const (
	DexType = "compound-v3"

	supplyGas   int64 = 150000
	withdrawGas int64 = 125000

	defaultReserve = 10000000000

	cometMethodIsWithdrawPaused = "isWithdrawPaused"
	cometMethodIsSupplyPaused   = "isSupplyPaused"
)
