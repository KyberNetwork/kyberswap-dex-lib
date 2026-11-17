package efraprop

// Config lists a single-maker AMM that quotes every listed token against USDC from prices
// the maker pushes on-chain.
type Config struct {
	DexID string `json:"dexID"`
	// Venue is the PropAmmContract: one instance holds every market, token <-> USDC.
	Venue string `json:"venue"`
	// Tokens are the listed stock tokens. Each becomes a pool against the venue's USDC(),
	// which is why only token <-> USDC pairs exist (the venue rejects token <-> token).
	Tokens []string `json:"tokens"`
	// Dest is the taker the venue keys gap discounts by: the address that will be msg.sender
	// of swapExactIn. Defaults to the KyberSwap executor.
	Dest string `json:"dest,omitempty"`
	// Buffer, in bps, haircuts every probed amountOut when set. The ladder interpolates
	// between probed points, so a small haircut covers the interpolation error.
	Buffer int64 `json:"buffer,omitempty"`
}

func (c *Config) dest() string {
	if c.Dest == "" {
		return defaultDest.Hex()
	}
	return c.Dest
}
