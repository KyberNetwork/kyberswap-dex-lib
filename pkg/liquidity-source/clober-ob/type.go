package cloberob

import (
	"github.com/ethereum/go-ethereum/common"

	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
)

type Liquidity struct {
	Tick  cloberlib.Tick `json:"tick"`
	Depth uint64         `json:"depth"`
}

type Extra struct {
	Highest cloberlib.Tick `json:"highest"`
	Depths  []Liquidity    `json:"depths"`
}

type StaticExtra struct {
	UnitSize    uint64              `json:"unitSize"`
	MakerPolicy cloberlib.FeePolicy `json:"makerPolicy"`
	TakerPolicy cloberlib.FeePolicy `json:"takerPolicy"`
	Hooks       common.Address      `json:"hooks"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SubgraphBook struct {
	Id                string `json:"id"`
	UnitSize          string `json:"unitSize"`
	MakerPolicy       string `json:"makerPolicy"`
	MakerFee          string `json:"makerFee"`
	IsMakerFeeInQuote bool   `json:"isMakerFeeInQuote"`
	TakerPolicy       string `json:"takerPolicy"`
	TakerFee          string `json:"takerFee"`
	IsTakerFeeInQuote bool   `json:"isTakerFeeInQuote"`
	Base              struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals string `json:"decimals"`
	} `json:"base"`
	Quote struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals string `json:"decimals"`
	} `json:"quote"`
	Hooks              string `json:"hooks"`
	Tick               string `json:"tick"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}
