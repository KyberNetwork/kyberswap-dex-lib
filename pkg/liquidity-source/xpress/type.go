package xpress

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type OrderBookLevels struct {
	ArrayPrices []*big.Int `json:"array_prices"`
	ArrayShares []*big.Int `json:"array_shares"`
}

type OrderBook struct {
	Bids OrderBookLevels `json:"bids"`
	Asks OrderBookLevels `json:"asks"`
}

type LobConfig struct {
	ScalingFactorTokenX           *big.Int       `json:"_scaling_factor_token_x"`
	ScalingFactorTokenY           *big.Int       `json:"_scaling_factor_token_y"`
	TokenX                        common.Address `json:"_token_x"`
	TokenY                        common.Address `json:"_token_y"`
	SupportsNativeEth             bool           `json:"_supports_native_eth"`
	IsTokenXWeth                  bool           `json:"_is_token_x_weth"`
	AskTrie                       common.Address `json:"_ask_trie"`
	BidTrie                       common.Address `json:"_bid_trie"`
	AdminCommissionRate           uint64         `json:"_admin_commission_rate"`
	TotalAggressiveCommissionRate uint64         `json:"_total_aggressive_commission_rate"`
	TotalPassiveCommissionRate    uint64         `json:"_total_passive_commission_rate"`
	PassiveOrderPayoutRate        uint64         `json:"_passive_order_payout_rate"`
	ShouldInvokeOnTrade           bool           `json:"_should_invoke_on_trade"`
}

type Metadata struct {
	Pools []common.Address `json:"pools"`
}

type TokenInfo struct {
	ContractAddress string `json:"contractAddress"`
	Decimals        uint8  `json:"decimals"`
	Symbol          string `json:"symbol"`
	IsNative        bool   `json:"isNative"`
}

type MarketInfo struct {
	OrderbookAddress     string    `json:"orderbookAddress"`
	BaseToken            TokenInfo `json:"baseToken"`
	QuoteToken           TokenInfo `json:"quoteToken"`
	TokenXScallingFactor uint8     `json:"tokenXScallingFactor"`
	TokenYScallingFactor uint8     `json:"tokenYScallingFactor"`
	AggressiveFee        float64   `json:"aggressiveFee"`
}

type SwapInfo struct {
	UpdatedOrderBook *OrderBook `json:"orderBook"`
}
