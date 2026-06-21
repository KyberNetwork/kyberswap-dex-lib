package machima

const (
	DexTypeMachima = "machima"

	// Pool fee is fixed at 1% (10000 bps) for all Machima pools
	PoolFee = 10000

	// Tick spacing for 1% fee tier
	TickSpacing = 200

	// Anti-sniper window in seconds (10 minutes)
	AntiSniperWindowSeconds = 600

	// Default gas estimates
	BaseGas       int64 = 350000
	CrossTickGas  int64 = 100000
)
