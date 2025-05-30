package hashflowv3

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPClientConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url"`
	Source     string                `mapstructure:"source" json:"source"`
	APIKey     string                `mapstructure:"api_key" json:"api_key"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count"`
	Client     *resty.Client
}

type QuoteParams struct {
	Source     string `json:"source"`
	BaseChain  Chain  `json:"baseChain"`
	QuoteChain Chain  `json:"quoteChain"`
	RFQs       []RFQ  `json:"rfqs"`
}

type Chain struct {
	ChainType string              `json:"chainType"`
	ChainId   valueobject.ChainID `json:"chainId"`
}

type RFQ struct {
	BaseToken           string      `json:"baseToken"`
	QuoteToken          string      `json:"quoteToken"`
	BaseTokenAmount     string      `json:"baseTokenAmount"`
	Trader              string      `json:"trader"`
	EffectiveTrader     string      `json:"effectiveTrader"`
	RewardTrader        string      `json:"rewardTrader"`
	MarketMakers        []string    `json:"marketMakers"`
	ExcludeMarketMakers []string    `json:"excludeMarketMakers"`
	Options             *RFQOptions `json:"options,omitempty"`
	FeesBps             uint        `json:"feesBps"`
}

type RFQOptions struct {
	DoNotRetryWithOtherMakers bool `json:"doNotRetryWithOtherMakers,omitempty"`
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
	Signature      string `json:"signature"`
	TargetContract string `json:"targetContract"`

	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type (
	PriceLevel struct {
		Quote float64
		Price float64
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
