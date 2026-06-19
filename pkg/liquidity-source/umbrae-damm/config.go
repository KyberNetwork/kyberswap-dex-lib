package umbraedamm

// Config drives the DAMM integration. DAMM has no factory/registry on-chain — each pair is a
// standalone constant-product contract — so the set of pools is supplied explicitly.
type Config struct {
	DexID string   `json:"dexID"`
	Pools []string `json:"pools"`
}
