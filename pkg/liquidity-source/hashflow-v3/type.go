package hashflowv3

type QuoteParams struct {
	Source     string `json:"source"`
	BaseChain  Chain  `json:"baseChain"`
	QuoteChain Chain  `json:"quoteChain"`
	RFQs       []RFQ  `json:"rfqs"`
}

type Chain struct {
	ChainType string `json:"chainType"`
	ChainId   uint   `json:"chainId"`
}

type RFQ struct {
	BaseToken           string   `json:"baseToken"`
	QuoteToken          string   `json:"quoteToken"`
	BaseTokenAmount     string   `json:"baseTokenAmount"`
	Trader              string   `json:"trader"`
	EffectiveTrader     string   `json:"effectiveTrader"`
	RewardTrader        string   `json:"rewardTrader"`
	MarketMakers        []string `json:"marketMakers"`
	ExcludeMarketMakers []string `json:"excludeMarketMakers"`
	FeesBps             uint     `json:"feesBps"`
}

type QuoteResult struct {
	Status string `json:"status"`
	Error  struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RfqId  string  `json:"rfqId"`
	Quotes []Quote `json:"quotes"`
}

type Quote struct {
	QuoteData struct {
		BaseChain  Chain `json:"baseChain"`
		QuoteChain Chain `json:"quoteChain"`

		BaseToken        string `json:"baseToken"`
		BaseTokenAmount  string `json:"baseTokenAmount"`
		QuoteToken       string `json:"quoteToken"`
		QuoteTokenAmount string `json:"quoteTokenAmount"`
		Trader           string `json:"trader"`
		EffectiveTrader  string `json:"effectiveTrader"`
		TxID             string `json:"txid"`
		Pool             string `json:"pool"`
		QuoteExpiry      int64  `json:"quoteExpiry"`
		Nonce            int64  `json:"nonce"`
		ExternalAccount  string `json:"externalAccount"`
	} `json:"quoteData"`
	Signature string `json:"signature"`
}
