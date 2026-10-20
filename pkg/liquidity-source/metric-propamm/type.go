package metricpropamm

import "math/big"

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
	Asks []PairBin `json:"asks"`
	Bids []PairBin `json:"bids"`
}

type PairBin struct {
	BinIdx           int64  `json:"binIdx"`
	Price            string `json:"price"`
	CummlativeVolume string `json:"cumulativeVolume"`
	PriceImpactE6    string `json:"priceImpactE6"`
}

type StaticExtra struct {
	Pair                 string `json:"pair"`
	PriceProviderAddress string `json:"priceProvider"`

	SwapWhitelistingEnabled bool `json:"swapWhitelisting,omitempty"`
}

type Extra struct {
	InitBid *big.Int `json:"initBid"`
	InitAsk *big.Int `json:"initAsk"`

	QuoteAvailable bool  `json:"qA"`
	MaxAge         int64 `json:"maxAge"`

	IsV2 bool `json:"isV2"`

	// Extensions is the number of pool extensions (metric-propamm), each adding swap gas.
	Extensions int `json:"ext,omitempty"`

	Asks []Bin `json:"asks"`
	Bids []Bin `json:"bids"`
}

type Bin struct {
	BinIdx           int64    `json:"bi"`
	Price            *big.Int `json:"p"`
	CumulativeVolume *big.Int `json:"cv"` // total amountOut that can be swapped up to this bin (inclusive)
}

type PoolMeta struct {
	SwapDirection        bool   `json:"swapDir"`
	PriceProviderAddress string `json:"priceProvider"`
	BlockNumber          uint64 `json:"blockNumber"`
}
