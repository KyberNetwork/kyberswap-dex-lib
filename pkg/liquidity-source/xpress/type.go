package xpress

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type OrderBookLevels struct {
	ArrayPrices []*uint256.Int `json:"p"`
	ArrayShares []*uint256.Int `json:"s"`
}

type OrderBookLevelsRPC struct {
	ArrayPrices []*big.Int
	ArrayShares []*big.Int
}

type OrderBook struct {
	Bids OrderBookLevels `json:"b"`
	Asks OrderBookLevels `json:"a"`
}

type OrderBookRPC struct {
	Bids OrderBookLevelsRPC
	Asks OrderBookLevelsRPC
}

type StaticExtra struct {
	ScalingFactorX    *uint256.Int `json:"sX"`
	ScalingFactorY    *uint256.Int `json:"sY"`
	SupportsNativeEth bool         `json:"eth"`
}

type LobConfig struct {
	ScalingFactorTokenX           *big.Int
	ScalingFactorTokenY           *big.Int
	ScalingFactorX                *uint256.Int
	ScalingFactorY                *uint256.Int
	TokenX                        common.Address
	TokenY                        common.Address
	SupportsNativeEth             bool
	IsTokenXWeth                  bool
	AskTrie                       common.Address
	BidTrie                       common.Address
	AdminCommissionRate           uint64
	TotalAggressiveCommissionRate uint64
	TotalPassiveCommissionRate    uint64
	PassiveOrderPayoutRate        uint64
	ShouldInvokeOnTrade           bool
}

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type TokenInfo struct {
	ContractAddress string `json:"contractAddress"`
	Decimals        uint8  `json:"decimals"`
	Symbol          string `json:"symbol"`
	IsNative        bool   `json:"isNative"`
}

type MarketInfo struct {
	OrderbookAddress string    `json:"orderbookAddress"`
	BaseToken        TokenInfo `json:"baseToken"`
	QuoteToken       TokenInfo `json:"quoteToken"`
	AggressiveFee    float64   `json:"aggressiveFee"`
}

type SwapInfo struct {
	UpdatedOrderBook *OrderBook `json:"orderBook"`
}
