package client

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type QuoteParams struct {
	// Origin address of the end trader
	Origin string `json:"origin"`
	// Sender address calling the Swaap vault
	Sender string `json:"sender"`
	// Recipient address receiving the funds
	Recipient string `json:"recipient"`
	// Timestamp of the request (in sec)
	Timestamp int64 `json:"timestamp"`
	// OrderType enum: 1 for SELL, 2 for BUY
	OrderType OrderType `json:"order_type"`
	// TokenIn address of the input token
	TokenIn string `json:"token_in"`
	// TokenOut address of the output token
	TokenOut string `json:"token_out"`
	// Amount input amount if SELL, output amount if BUY. fixed-point format
	Amount string `json:"amount"`
	// Tolerance price tolerance. should be > 0 or will be mapped to 0.01618034 (1.618034%)
	Tolerance float64 `json:"tolerance,omitempty"`
	// NetworkID e.g. 1 for Ethereum, 137 for Polygon
	NetworkID valueobject.ChainID `json:"network_id"`
	// ReferralFees e.g. 0.0001 for 1bps
	ReferralFees float64 `json:"referral_fees"`
	// Authorizer NB: Only for solvers
	Authorizer string `json:"authorizer,omitempty"`
}

type QuoteResult struct {
	// ID of the transaction
	ID string `json:"id"`
	// Recipient address receiving the funds
	Recipient string `json:"recipient"`
	// Expiration expiration timestamp
	Expiration int64 `json:"expiration"`
	// Amount quote amount
	Amount string `json:"amount"`
	// ExpectedPrice will be met or revert
	ExpectedPrice float64 `json:"expected_price"`
	// GuaranteedPrice will be met or revert
	GuaranteedPrice float64 `json:"guaranteed_price"`
	Success         bool    `json:"success"`
	// Calldata to execute onchain
	Calldata string `json:"calldata"`
	// Router Swaap router address
	Router string `json:"router"`
}

type OrderType int

var (
	OrderTypeSell OrderType = 1
	OrderTypeBuy  OrderType = 2
)
