package saturnusdat

type Config struct {
	DexID string `json:"dexID"`
	// Executor is the caller that mints and receives USDat. When set, the tracker reads its
	// whitelist status so quotes stop while USDat would revert with AccountNotWhitelisted.
	Executor string `json:"executor"`
}
