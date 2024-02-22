package hashflowv3

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"baseUrl"`
	Source     string                `mapstructure:"source" json:"source"`
	APIKey     string                `mapstructure:"api_key" json:"apiKey"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retryCount"`
}

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
		QuoteData struct {
			BaseChain  Chain `json:"baseChain"`
			QuoteChain Chain `json:"quoteChain"`
		}
		BaseToken        string `json:"baseToken"`
		BaseTokenAmount  string `json:"baseTokenAmount"`
		QuoteToken       string `json:"quoteToken"`
		QuoteTokenAmount string `json:"quoteTokenAmount"`
		Trader           string `json:"trader"`
		TxID             string `json:"txid"`
		Pool             string `json:"pool"`
		QuoteExpiry      string `json:"quoteExpiry"`
		Nonce            string `json:"nonce"`
		Signature        string `json:"signature"`
	} `json:"quotes"`
}
