package dexalot

import (
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
	return !r.Success || r.ReasonCode != "" || r.Reason != ""
}

type Order struct {
	NonceAndMeta string `json:"nonceAndMeta"`
	Expiry       int64  `json:"expiry"`
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
