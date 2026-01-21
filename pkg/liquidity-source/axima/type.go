package axima

type PairMetadata struct {
	Pair                 string `json:"pair"`
	PoolAddress          string `json:"poolAddress"`
	PriceProviderAddress string `json:"priceProviderAddress"`
	QuoterAddress        string `json:"quoterAddress"`
	Token0               string `json:"token0"`
	Token1               string `json:"token1"`
}

type PairData struct {
	Bid                  string `json:"bidAdj"`
	Ask                  string `json:"askAdj"`
	QuoteAvailable       bool   `json:"quoteAvailable"`
	TotalToken0Available string `json:"totalToken0Available"`
	TotalToken1Available string `json:"totalToken1Available"`
	QuoteExpiration      int64  `json:"quoteExpiration"`
}

type StaticExtra struct {
	Pair string `json:"pair"`
}

type Extra struct {
	ZeroToOneRate   float64 `json:"0to1R"`
	OneToZeroRate   float64 `json:"1to0R"`
	QuoteAvailable  bool    `json:"qA"`
	QuoteExpiration int64   `json:"qE"`
}
