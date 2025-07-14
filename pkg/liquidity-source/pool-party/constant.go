package poolparty

const (
	DexType = "pool-party"

	graphFirstLimit = 500

	poolStatusCanceled = "CANCELED"
	poolStatusActive   = "ACTIVE"

	defaultGas = 2_500_000 // From their docs: ~200_000 - 400_000 gas. But actual simulation shows it's costly.
)
