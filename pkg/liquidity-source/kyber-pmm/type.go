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
	Groups   map[string][]string  `json:"groups"`
}

type StaticExtra struct {
	PairIDs             []string `json:"pairIDs"`
	BaseTokenAddress    string   `json:"baseTokenAddress"`
	QuoteTokenAddresses []string `json:"quoteTokenAddress"`
}

type Extra struct {
	PriceLevels map[string]BaseQuotePriceLevels `json:"priceLevels"` // base-quote -> price_levels
}

type BaseQuotePriceLevels struct {
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

type Order struct {
	MakerAsset          string `json:"maker_asset"`
	TakerAsset          string `json:"taker_asset"`
	TakerAmount         string `json:"taker_amount"`
	ExpectedMakerAmount string `json:"expected_maker_amount"`
	MinMakerAmount      string `json:"min_maker_amount"`
}

type MultiFirmRequestParams struct {
	RequestID   string  `json:"request_id"`
	Orders      []Order `json:"orders"`
	UserAddress string  `json:"user_address"`
	RFQSender   string  `json:"rfq_sender"`
	Partner     string  `json:"partner"`
}

type FirmRequestParams struct {
	MakerAsset  string `json:"makerAsset"`
	TakerAsset  string `json:"takerAsset"`
	MakerAmount string `json:"makerAmount"`
	TakerAmount string `json:"takerAmount"`
	UserAddress string `json:"userAddress"`
	RFQSender   string `json:"rfqSender"`
}

type MultiFirmResult struct {
	Orders []struct {
		MakerAsset  string `json:"maker_asset"`
		TakerAsset  string `json:"taker_asset"`
		MakerAmount string `json:"maker_amount"`
		TakerAmount string `json:"taker_amount"`
		FeeAmount   string `json:"fee_amount"`
		Signature   string `json:"signature"`
		Error       string `json:"error,omitempty"`
	}

	Info          string `json:"info"`
	Expiry        int64  `json:"expiry"`
	Maker         string `json:"maker"`
	Taker         string `json:"taker"`
	AllowedSender string `json:"allowed_sender"`
	Error         string `json:"error,omitempty"`
}

type FirmResult struct {
	Order struct {
		Info          string `json:"info"`
		Expiry        int64  `json:"expiry"`
		MakerAsset    string `json:"makerAsset"`
		TakerAsset    string `json:"takerAsset"`
		Maker         string `json:"maker"`
		Taker         string `json:"taker"`
		MakerAmount   string `json:"makerAmount"`
		TakerAmount   string `json:"takerAmount"`
		Signature     string `json:"signature"`
		AllowedSender string `json:"allowedSender"`
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
	AllowedSender      string `json:"allowedSender"`
	Partner            string `json:"partner"`
	QuoteTimestamp     int64  `json:"quoteTimestamp"`
}

type RFQMeta struct {
	Timestamp int64 `json:"timestamp"`
}
