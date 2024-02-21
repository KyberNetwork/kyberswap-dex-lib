package hashflowv3

type QuoteParams struct {
	Source    string               `json:"source"`
	BaseChain QuoteParamsBaseChain `json:"baseChain"`
	RFQs      []QuoteParamsRFQ     `json:"rfqs"`
}

type QuoteParamsBaseChain struct {
	ChainType string `json:"chainType"`
	ChainId   uint   `json:"chainId"`
}

type QuoteParamsRFQ struct {
	BaseToken           string   `json:"baseToken"`
	QuoteToken          string   `json:"quoteToken"`
	BaseTokenAmount     string   `json:"baseTokenAmount"`
	QuoteTokenAmount    string   `json:"quoteTokenAmount"`
	Trader              string   `json:"trader"`
	EffectiveTrader     string   `json:"effectiveTrader"`
	RewardTrader        string   `json:"rewardTrader"`
	MarketMakers        []string `json:"marketMakers"`
	ExcludeMarketMakers []string `json:"excludeMarketMakers"`
	FeesBps             uint     `json:"feesBps"`
}

type QuoteResult struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	RfqId  string `json:"rfqId"`
	Quotes []struct {
	} `json:"quotes"`
}
