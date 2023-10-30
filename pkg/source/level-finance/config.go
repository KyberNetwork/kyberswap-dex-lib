package levelfinance

type Config struct {
	DexID                string `json:"dexID"`
	LiquidityPoolAddress string `json:"liquidityPoolAddress"`
	ChainID              int    `json:"chainID"`
}
