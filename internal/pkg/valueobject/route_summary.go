package valueobject

import (
	"encoding/binary"
	"math/big"

	"github.com/cespare/xxhash/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

// RouteSummary contains route and summarized data around the route such as gas, amounts in USD,...
type RouteSummary struct {
	// TokenIn address of token to be swapped
	TokenIn string `json:"tokenIn"`

	// AmountIn  amount of token to be swapped
	AmountIn *big.Int `json:"amountIn"`

	// AmountInUSD amount in USD to be swapped
	AmountInUSD float64 `json:"amountInUsd"`

	// TokenOut address of token to be received
	TokenOut string `json:"tokenOut"`

	// AmountOut amount of token to be received
	AmountOut *big.Int `json:"amountOut"`

	// AmountOutUSD amount in USD of token to be received
	AmountOutUSD float64 `json:"amountOutUsd"`

	// Gas total gas consumed for swapping
	Gas int64 `json:"gas"`

	// GasPrice price of gas (in Wei)
	GasPrice *big.Float `json:"gasPrice"`

	// GasUSD gas in USD
	GasUSD float64 `json:"gasUsd"`

	// L1FeeUSD L1 fee in USD (for some L2 chains)
	L1FeeUSD float64 `json:"l1FeeUsd"`

	// ExtraFee extra fee should be charged when executing swap, can be customized by client
	ExtraFee ExtraFee `json:"extraFee"`

	// Alpha fee
	AlphaFee *entity.AlphaFeeV2 `json:"-"`

	// Route
	Route [][]Swap `json:"route"`

	// RouteID request_id from GET routes request
	RouteID string `json:"routeID"`
	// Timestamp of the GET routes request
	Timestamp int64 `json:"timestamp"`
	// OriginalChecksum of the GET routes request
	OriginalChecksum uint64 `json:"checksum,string"`
}

// Checksum only uses enough data to avoid "return amount not enough" due to manually modify amount out and swap amount
func (rs RouteSummary) Checksum(salt string) *xxhash.Digest {
	h := xxhash.New()
	_, _ = h.WriteString(salt)
	_, _ = h.WriteString(rs.TokenIn)
	_, _ = h.Write(rs.AmountIn.Bytes())

	_, _ = h.WriteString(rs.TokenOut)
	_, _ = h.Write(rs.AmountOut.Bytes())
	_, _ = h.Write(binary.LittleEndian.AppendUint64(nil, uint64(rs.Timestamp)))

	_, _ = h.WriteString(rs.RouteID)

	// Add alpha fee to checksum because we want to limit the calls to Redis.
	// In case routeSummary doesn't have alpha fee and the routeSummary hasn't been modified
	// checksum validation always return true, and we don't need to retrieve checksum from Redis.
	if rs.AlphaFee != nil {
		for _, swapReduction := range rs.AlphaFee.SwapReductions {
			_, _ = h.Write(binary.LittleEndian.AppendUint64(nil, uint64(swapReduction.ExecutedId)))
			_, _ = h.Write(swapReduction.ReduceAmount.Bytes())
		}
	}

	for _, path := range rs.Route {
		for _, swap := range path {
			_, _ = h.WriteString(swap.Pool)
			_, _ = h.WriteString(swap.TokenIn)
			_, _ = h.WriteString(swap.TokenOut)
			_, _ = h.Write(swap.SwapAmount.Bytes())
			_, _ = h.Write(swap.AmountOut.Bytes())
		}
	}

	return h
}

func (rs RouteSummary) GetPriceImpact() float64 {
	return (rs.AmountInUSD - rs.AmountOutUSD) / rs.AmountInUSD
}

type RouteSummaries struct {
	BestRoute                *RouteSummary
	AMMBestRoute             *RouteSummary
	BestRouteBeforeMergeSwap *RouteSummary
}

const (
	BestRoute int = iota
	AMMBestRoute
)

func (b RouteSummaries) GetBestRouteSummary() *RouteSummary {
	return b.BestRoute
}

func (b RouteSummaries) GetAMMBestRouteSummary() *RouteSummary {
	return b.AMMBestRoute
}

func (b RouteSummaries) GetBestRouteBeforeMergeSwap() *RouteSummary {
	return b.BestRouteBeforeMergeSwap
}
