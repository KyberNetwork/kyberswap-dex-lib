package clipper

import "math/big"

type Extra struct {
	ChainID       uint
	SwapsEnabled  bool
	K             float64
	TimeInSeconds int
	Assets        []PoolAsset
	Pairs         []PoolPair
}

type PoolAsset struct {
	Address       string
	Symbol        string
	Decimals      uint8
	PriceInUSD    float64
	Quantity      *big.Int
	ListingWeight int
}

type PoolPair struct {
	Assets           [2]string
	FeeInBasisPoints float64
}

type SwapInfo struct {
	ChainID           uint
	TimeInSeconds     int
	InputAmount       *big.Int
	InputAssetSymbol  string
	OutputAssetSymbol string
}

type QuoteParams struct {
	ChainID           uint   `json:"chain_id"`
	TimeInSeconds     int    `json:"time_in_seconds"`
	InputAmount       string `json:"input_amount"`
	InputAssetSymbol  string `json:"input_asset_symbol"`
	OutputAssetSymbol string `json:"output_asset_symbol"`

	DestinationAddress string // use in sign request
	SenderAddress      string // use in sign request
}

type QuoteResponse struct {
	ID           string `json:"id"`
	GoodUntil    int    `json:"good_until"`
	OutputAmount string `json:"output_amount"`
}

type SignParams struct {
	QuoteID            string `json:"quote_id"`
	DestinationAddress string `json:"destination_address"`
	SenderAddress      string `json:"sender_address"`
	NativeInput        bool   `json:"native_input"`
	NativeOutput       bool   `json:"native_output"`
}

type SignResponse struct {
	OutputAmount string `json:"output_amount"`
	Signature    struct {
		V int    `json:"v"`
		R string `json:"r"`
		S string `json:"s"`
	} `json:"signature"`
}
