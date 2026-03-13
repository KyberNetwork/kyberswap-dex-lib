package wcm

import (
	"math/big"
)

type Gas struct {
	Base  int64
	Level int64
}

type OrderBookLevel struct {
	Price    *big.Int `json:"price"`
	Quantity *big.Int `json:"quantity"`
}

type OrderBook struct {
	Bids []OrderBookLevel `json:"bids"`
	Asks []OrderBookLevel `json:"asks"`
}

type StaticExtra struct {
	Router                   string `json:"router"`
	BuyTokenPositionDecimals uint8  `json:"buyTokenPositionDecimals,omitempty"`
	PayTokenPositionDecimals uint8  `json:"payTokenPositionDecimals,omitempty"`
}

type Extra struct {
	OrderBook          OrderBook `json:"orderBook"`
	MinOrderQuantity   *big.Int  `json:"minOrderQuantity"`
	TakerFeeMultiplier *big.Int  `json:"takerFeeMultiplier"`
	FromMaxFee         *big.Int  `json:"fromMaxFee"`
	ToMaxFee           *big.Int  `json:"toMaxFee"`
	IsHalted           bool      `json:"isHalted"`
}

type SwapInfo struct {
	IsBuy          bool     `json:"isBuy"`
	ExecutedLevels int      `json:"executedLevels"`
	GrossBase      *big.Int `json:"grossBase"`
	GrossQuote     *big.Int `json:"grossQuote"`
}

type Metadata struct {
	// (min(tokenId1,tokenId2)<<32) | max(tokenId1,tokenId2)
	SeenPairKeys []uint64 `json:"seenPairKeys"`
}
