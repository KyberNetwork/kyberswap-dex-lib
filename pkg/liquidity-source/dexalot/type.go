package dexalot

import (
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/mitchellh/mapstructure"
)

type QueryParams = string

const (
	ParamsChainID     QueryParams = "chainid"
	ParamsTakerAsset  QueryParams = "takerAsset"
	ParamsMakerAsset  QueryParams = "makerAsset"
	ParamsTakerAmount QueryParams = "takerAmount"
	ParamsMakerAmount QueryParams = "makerAmount"
	ParamsUserAddress QueryParams = "userAddress"
	ParamsExecutor    QueryParams = "executor"
	ParamsSlippage    QueryParams = "slippage"
	ParamsPartner     QueryParams = "partner"
	ParamsTxType      QueryParams = "txType"
)

type FirmQuoteParams struct {
	ChainID     int    `mapstructure:"chainid"`
	TakerAsset  string `mapstructure:"takerAsset"`
	MakerAsset  string `mapstructure:"makerAsset"`
	TakerAmount string `mapstructure:"takerAmount"`
	UserAddress string `mapstructure:"userAddress"`
	Executor    string `mapstructure:"executor"`
	Partner     string `mapstructure:"partner"`
}

func (p *FirmQuoteParams) ToMap() (ret map[string]string) {
	if err := mapstructure.Decode(p, &ret); err != nil {
		logger.WithFields(logger.Fields{"params": p, "error": err}).Error("failed to decode to map")
	}
	return ret
}

type FirmQuoteResult struct {
	Order     Order       `json:"order"`
	Signature string      `json:"signature"`
	Tx        Transaction `json:"tx"`
}
type FirmQuoteFail struct {
	Success    bool   `json:"Success"`
	ReasonCode string `json:"ReasonCode"`
	Reason     string `json:"Reason"`
}

func (r FirmQuoteFail) Failed() bool {
	return !r.Success
}

type Order struct {
	NonceAndMeta string `json:"nonceAndMeta"`
	Expiry       int    `json:"expiry"`
	MakerAsset   string `json:"makerAsset"`
	TakerAsset   string `json:"takerAsset"`
	Maker        string `json:"maker"`
	Taker        string `json:"taker"`
	MakerAmount  string `json:"makerAmount"`
	TakerAmount  string `json:"takerAmount"`
}

type Transaction struct {
	To       string `json:"to"`
	Data     string `json:"data"`
	GasLimit int    `json:"gasLimit"`
}

type (
	SwapInfo struct {
		BaseToken          string `json:"b" mapstructure:"b"`
		BaseTokenAmount    string `json:"bAmt" mapstructure:"bAmt"`
		QuoteToken         string `json:"q" mapstructure:"q"`
		QuoteTokenAmount   string `json:"qAmt" mapstructure:"qAmt"`
		MarketMaker        string `json:"mm,omitempty" mapstructure:"mm"`
		ExpirySecs         uint   `json:"exp,omitempty" mapstructure:"exp"`
		BaseTokenOriginal  string `json:"bo,omitempty" mapstructure:"bo"`
		QuoteTokenOriginal string `json:"qo,omitempty" mapstructure:"qo"`
		BaseTokenReserve   string `json:"br,omitempty" mapstructure:"br"`
		QuoteTokenReserve  string `json:"qr,omitempty" mapstructure:"qr"`
	}

	Gas struct {
		Quote int64
	}

	PriceLevel struct {
		Quote *big.Float
		Price *big.Float
	}

	PriceLevelRaw struct {
		Price float64 `json:"p"`
		Quote float64 `json:"q"`
	}

	Extra struct {
		ZeroToOnePriceLevels []PriceLevelRaw `json:"0to1"`
		OneToZeroPriceLevels []PriceLevelRaw `json:"1to0"`
		Token0Address        string          `json:"token0"`
		Token1Address        string          `json:"token1"`
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)
