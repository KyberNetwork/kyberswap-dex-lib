package hashflowv3

import "time"

type HTTPClientConfig struct {
	BaseURL    string        `mapstructure:"base_url" json:"baseUrl"`
	Source     string        `mapstructure:"source" json:"source"`
	APIKey     string        `mapstructure:"api_key" json:"apiKey"`
	Timeout    time.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int           `mapstructure:"retry_count" json:"retryCount"`
}

type QuoteParams struct {
	Source     string     `json:"source"`
	BaseChain  QuoteChain `json:"baseChain"`
	QuoteChain QuoteChain `json:"quoteChain"`
	RFQs       []QuoteRFQ `json:"rfqs"`
}

type QuoteChain struct {
	ChainType string `json:"chainType"`
	ChainId   uint   `json:"chainId"`
}

type QuoteRFQ struct {
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
