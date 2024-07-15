package valueobject

import (
	"math/big"
)

// RouteSummary contains route and summarized data around the route such as gas, amounts in USD,...
type RouteSummary struct {
	// TokenIn address of token to be swapped
	TokenIn string `json:"tokenIn"`

	// AmountIn  amount of token to be swapped
	AmountIn *big.Int `json:"amountIn"`

	// AmountInUSD amount in USD to be swapped
	AmountInUSD float64 `json:"amountInUsd"`

	// TokenInMarketPriceAvailable indicate if token in has market price or not
	TokenInMarketPriceAvailable bool `json:"tokenInMarketPriceAvailable"`

	// TokenOut address of token to be received
	TokenOut string `json:"tokenOut"`

	// AmountOut amount of token to be received
	AmountOut *big.Int `json:"amountOut"`

	// AmountOutUSD amount in USD of token to be received
	AmountOutUSD float64 `json:"amountOutUsd"`

	// TokenOutMarketPriceAvailable indicate if token out has market price or not
	TokenOutMarketPriceAvailable bool `json:"tokenOutMarketPriceAvailable"`

	// Gas total gas consumed for swapping
	Gas int64 `json:"gas"`

	// GasPrice price of gas (in Wei)
	GasPrice *big.Float `json:"gasPrice"`

	// GasUSD gas in USD
	GasUSD float64 `json:"gasUsd"`

	// ExtraFee extra fee should be charged when executing swap, can be customized by client
	ExtraFee ExtraFee `json:"extraFee"`

	// Route
	Route [][]Swap `json:"route"`
}

func (rs RouteSummary) GetPriceImpact() float64 {
	return (rs.AmountInUSD - rs.AmountOutUSD) / rs.AmountInUSD
}
