package kyberpmm

type TokenItem struct {
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Decimals    uint8  `json:"decimals"`
	Type        string `json:"type"`
}

// ListTokensResult is the result of list tokens
type ListTokensResult struct {
	Tokens map[string]TokenItem `json:"tokens"`
}

type PairItem struct {
	Base  string `json:"base"`
	Quote string `json:"quote"`

	// LiquidityUSD fetched from API is very small, so we only keep track it, not use it for now
	LiquidityUSD float64 `json:"liquidityUSD"`
}

// ListPairsResult is the result of list pairs
type ListPairsResult struct {
	Pairs map[string]PairItem `json:"pairs"`
}

type PriceItem struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

// ListPriceLevelsResult is the result of list price levels
type ListPriceLevelsResult struct {
	Prices map[string]PriceItem `json:"prices"`
}

type StaticExtra struct {
	PairID            string `json:"pairID"`
	BaseTokenAddress  string `json:"baseTokenAddress"`
	QuoteTokenAddress string `json:"quoteTokenAddress"`
}

type Extra struct {
	BaseToQuotePriceLevels []PriceLevel `json:"baseToQuotePriceLevels"`
	QuoteToBasePriceLevels []PriceLevel `json:"quoteToBasePriceLevels"`
}

type PriceLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type SwapInfo struct {
	BaseToken        string `json:"baseToken"`
	BaseTokenAmount  string `json:"baseTokenAmount"`
	QuoteToken       string `json:"quoteToken"`
	QuoteTokenAmount string `json:"quoteTokenAmount"`
}

type Gas struct {
	Swap int64
}
