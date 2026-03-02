package axima

import "math/big"

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
	Depth                Depth  `json:"depth"`
}

type Depth struct {
	Asks []AximaBin `json:"asks"`
	Bids []AximaBin `json:"bids"`
}

type AximaBin struct {
	BinIdx           int64  `json:"binIdx"`
	Price            string `json:"price"`
	CummlativeVolume string `json:"cumulativeVolume"`
	PriceImpactE6    string `json:"priceImpactE6"`
}

type StaticExtra struct {
	Pair string `json:"pair"`
}

type Extra struct {
	QuoteAvailable bool  `json:"qA"`
	MaxAge         int64 `json:"maxAge"`

	Asks []Bin `json:"asks"`
	Bids []Bin `json:"bids"`
}

type Bin struct {
	BinIdx           int64    `json:"bi"`
	Rate             float64  `json:"r"`
	CumulativeVolume *big.Int `json:"cv"` // total amountOut that can be swapped up to this bin (inclusive)
	PriceImpactE6    int      `json:"pie6"`
}
