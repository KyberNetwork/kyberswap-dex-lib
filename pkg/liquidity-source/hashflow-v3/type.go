package hashflowv3

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type HTTPClientConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url"`
	Source     string                `mapstructure:"source" json:"source"`
	APIKey     string                `mapstructure:"api_key" json:"api_key"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count"`
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

type (
	PriceLevel struct {
		Quote *big.Float
		Price *big.Float
	}

	StaticExtra struct {
		MarketMaker string `json:"marketMaker"`
	}

	Extra struct {
		ZeroToOnePriceLevels []PriceLevelRaw `json:"zeroToOnePriceLevels"`
		OneToZeroPriceLevels []PriceLevelRaw `json:"oneToZeroPriceLevels"`
		PriceTolerance       int64           `json:"priceTolerance"`
	}
	PriceLevelRaw struct {
		Quote string `json:"q"`
		Price string `json:"p"`
	}

	SwapInfo struct {
		BaseToken        string `json:"baseToken"`
		BaseTokenAmount  string `json:"baseTokenAmount"`
		QuoteToken       string `json:"quoteToken"`
		QuoteTokenAmount string `json:"quoteTokenAmount"`
		MarketMaker      string `json:"marketMaker"`
	}

	Gas struct {
		Quote int64
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)
