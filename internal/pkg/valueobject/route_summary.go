package valueobject

import (
	"encoding/binary"
	"math/big"

	"github.com/cespare/xxhash/v2"
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

	Timestamp int64 `json:"timestamp"`
}

// Only use enough data to avoid "return amount not enough" due to manually modify amount out and swap amount
func (rs RouteSummary) Checksum(salt string) *xxhash.Digest {
	h := xxhash.New()
	h.WriteString(salt)
	h.WriteString(rs.TokenIn)
	h.Write(rs.AmountIn.Bytes())

	h.WriteString(rs.TokenOut)
	h.Write(rs.AmountOut.Bytes())
	_, _ = h.Write(binary.LittleEndian.AppendUint64(nil, uint64(rs.Timestamp)))

	for _, path := range rs.Route {
		for _, swap := range path {
			h.WriteString(swap.Pool)
			h.WriteString(swap.TokenIn)
			h.WriteString(swap.TokenOut)
			h.Write(swap.SwapAmount.Bytes())
			h.Write(swap.AmountOut.Bytes())
		}
	}

	return h
}

func (rs RouteSummary) GetPriceImpact() float64 {
	return (rs.AmountInUSD - rs.AmountOutUSD) / rs.AmountInUSD
}
