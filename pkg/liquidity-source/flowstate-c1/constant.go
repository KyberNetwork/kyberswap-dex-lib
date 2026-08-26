package flowstatec1

const (
	DexType = "flowstate-c1"

	defaultGas = 250_000

	// defaultProbeAmount is the token amount used to sample quoteBuyFromPool for a unit
	// rate. Pricing has no curve impact up to fillableAmount (vendor spec section 3.1),
	// so any amount well under expected depth yields the same per-unit rate.
	defaultProbeAmount = "1000000000000000" // 1e15
)
