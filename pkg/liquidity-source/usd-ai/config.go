package usd_ai

// Config holds chain-specific addresses for the USDai converter.
// Pool address = USDaiAddress; token0 = USDai, token1 = BaseTokenAddress (PYUSD).
// Token info (decimals, symbol, ...) is filled by downstream after GetNewPools and stored in Redis; tracker/simulator read from entity.Pool.
type Config struct {
	// USDaiAddress is the USDai contract address (pool address).
	USDaiAddress string `json:"usdaiAddress"`
	// BaseTokenAddress is the base token (e.g. PYUSD) address.
	BaseTokenAddress string `json:"baseTokenAddress"`
}
