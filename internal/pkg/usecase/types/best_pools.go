package types

// BestPools contains best scoring pool ids
type BestPools struct {
	// PoolIds pools with tokenIn-tokenOut swap
	PoolIds []string

	// WhitelistPoolIds pools with whitelist-whitelist swap
	WhitelistPoolIds []string

	// TokenInPoolIds pools with whitelist-tokenIn swap
	TokenInPoolIds []string

	// TokenOutPoolIds pools with whitelist-tokenOut swap
	TokenOutPoolIds []string
}
