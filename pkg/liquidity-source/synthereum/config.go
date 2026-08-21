package synthereum

type Config struct {
	DexID string `json:"dexID"`
	// Finder is the SynthereumFinder for the chain: the single immutable anchor the
	// whole deployment is discovered from. It resolves the PoolRegistry and
	// FixedRateRegistry, which enumerate every pool, and each pool then reports its
	// own tokens (and the wrapper its lending vault). Nothing else needs configuring,
	// and new deployments are picked up without a config or library change.
	Finder string `json:"finder"`
}
