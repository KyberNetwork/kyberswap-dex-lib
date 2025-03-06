package entity

import "math/big"

type RFQRouteMessage struct {
	RouteID         string   `json:"route_id"`
	RFQSource       string   `json:"rfq_source"`
	QuoteTimestamp  int64    `json:"quote_timestamp"`
	SellToken       string   `json:"sell_token"`
	BuyToken        string   `json:"buy_token"`
	RequestedAmount *big.Int `json:"requested_amount"`
	TakerAmount     *big.Int `json:"taker_amount"`
	MakerAmount     *big.Int `json:"maker_amount"`
	TakerAsset      string   `json:"taker_asset"`
	MakerAsset      string   `json:"maker_asset"`
	AmmAmount       *big.Int `json:"amm_amount"`
	AlphaFee        *big.Int `json:"alpha_fee"`
	AlphaFeeInUSD   float64  `json:"alpha_fee_in_usd"`
	Partner         string   `json:"partner"`
	RouteType       string   `json:"route_type"`
}
