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
	Prices   map[string]PriceItem `json:"prices"`
	Balances map[string]float64   `json:"balances"`
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

type SwapExtra struct {
	TakerAsset   string `json:"takerAsset"`
	TakingAmount string `json:"takingAmount"`
	MakerAsset   string `json:"makerAsset"`
	MakingAmount string `json:"makingAmount"`
}

type Gas struct {
	Swap int64
}

type FirmRequestParams struct {
	MakerAsset  string `json:"makerAsset"`
	TakerAsset  string `json:"takerAsset"`
	MakerAmount string `json:"makerAmount"`
	TakerAmount string `json:"takerAmount"`
	UserAddress string `json:"userAddress"`
}

type FirmResult struct {
	Order struct {
		Info        string `json:"info"`
		Expiry      int64  `json:"expiry"`
		MakerAsset  string `json:"makerAsset"`
		TakerAsset  string `json:"takerAsset"`
		Maker       string `json:"maker"`
		Taker       string `json:"taker"`
		MakerAmount string `json:"makerAmount"`
		TakerAmount string `json:"takerAmount"`
		Signature   string `json:"signature"`
	} `json:"order"`

	Error string `json:"error"`
}

type RFQExtra struct {
	RFQContractAddress string `json:"rfqContractAddress"`
	Info               string `json:"info"`
	Expiry             int64  `json:"expiry"`
	MakerAsset         string `json:"makerAsset"`
	TakerAsset         string `json:"takerAsset"`
	Maker              string `json:"maker"`
	Taker              string `json:"taker"`
	MakerAmount        string `json:"makerAmount"`
	TakerAmount        string `json:"takerAmount"`
	Signature          string `json:"signature"`
	Recipient          string `json:"recipient"`
}

type RFQMeta struct {
	Timestamp int64 `json:"timestamp"`
}
