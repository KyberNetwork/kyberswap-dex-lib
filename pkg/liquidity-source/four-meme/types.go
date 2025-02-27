package fourmeme

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type TokenInfo struct {
	Version        *big.Int       `json:"version"`        // The TokenManager version. If version returns 1, you should call V1 TokenManager methods for trading. If version returns 2, call V2
	TokenManager   common.Address `json:"tokenManager"`   // The address of the token manager which manages your token. We recommend using this address to call the TokenManager-related interfaces and parameters, replacing the hardcoded TokenManager addresses
	Quote          common.Address `json:"quote"`          // The address of the quote token of your token. If quote returns address 0, it means the token is traded by BNB. otherwise traded by BEP20
	LastPrice      *big.Int       `json:"lastPrice"`      // The last price of your token
	TradingFeeRate *big.Int       `json:"tradingFeeRate"` // The trading fee rate of your token. The actual usage of the fee rate should be the return value divided by 10,000
	MinTradingFee  *big.Int       `json:"minTradingFee"`  // The amount of minimum trading fee
	LaunchTime     *big.Int       `json:"launchTime"`     // Launch time of the token
	Offers         *big.Int       `json:"offers"`         // Amount of tokens that are not sold
	MaxOffers      *big.Int       `json:"maxOffers"`      // Maximum amount of tokens that could be sold before creating Pancake pair
	Funds          *big.Int       `json:"funds"`          // Amount of paid BNB or BEP20 received
	MaxFunds       *big.Int       `json:"maxFunds"`       // Maximum amount of paid BNB or BEP20 that could be received
	LiquidityAdded bool           `json:"liquidityAdded"` // True if the Pancake pair has been created
}

type TokenData struct {
	Token           common.Address `json:"token"`
	RaisedToken     common.Address `json:"tokenManager"`
	TemplateId      *big.Int       `json:"templateId"`
	Field3          *big.Int       `json:"field3"`
	MaxOffers       *big.Int       `json:"maxOffers"`
	MaxFunds        *big.Int       `json:"maxFunds"`
	LaunchTime      *big.Int       `json:"launchTime"`
	Offers          *big.Int       `json:"offers"`
	Funds           *big.Int       `json:"funds"`
	Price           *big.Int       `json:"price"`
	Field10         *big.Int       `json:"field10"`
	Field11         *big.Int       `json:"field11"`
	TradingDisabled *big.Int       `json:"tradingDisabled"`
}

type FeeInfo struct {
	TokenTxFee *big.Int
	_          *big.Int
	_          *big.Int
	_          *big.Int
	_          *big.Int
}

type StaticExtra struct {
	BondingAddress string `json:"bondingAddress"`
}

type Extra struct {
	GradThreshold  *big.Int `json:"gradThreshold"`
	KLast          *big.Int `json:"kLast"`
	BuyTax         *big.Int `json:"buyTax"`
	SellTax        *big.Int `json:"sellTax"`
	ReserveA       *big.Int `json:"reserveA"`
	ReserveB       *big.Int `json:"reserveB"`
	TradingFeeRate *big.Int `json:"tradingFeeRate"`
	LaunchTime     *big.Int `json:"launchTime"`
}

type SwapInfo struct {
	TradedAmount *uint256.Int `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Gas struct {
	Swap int64
}
