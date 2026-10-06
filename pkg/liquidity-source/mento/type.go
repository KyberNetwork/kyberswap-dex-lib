package mento

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// StaticExtra holds immutable pool metadata captured at discovery.
type StaticExtra struct {
	Decimals0 string `json:"dec0"` // 10**token0.decimals(), as FPMM stores it
	Decimals1 string `json:"dec1"` // 10**token1.decimals()
}

// TradingLimit mirrors ITradingLimitsV2.Config + State for one token.
// Limits and netflows are in 15-decimal units (INTERNAL_DECIMALS).
type TradingLimit struct {
	Limit0       *int256.Int `json:"l0"`
	Limit1       *int256.Int `json:"l1"`
	Decimals     uint8       `json:"dec"`
	LastUpdated0 uint32      `json:"lu0"`
	LastUpdated1 uint32      `json:"lu1"`
	Netflow0     *int256.Int `json:"nf0"`
	Netflow1     *int256.Int `json:"nf1"`
}

// Extra is rewritten on every tracker refresh. Every field mirrors a check
// on the on-chain swap path (FPMM.swap -> OracleAdapter.getFXRateIfValid ->
// TradingLimitsV2).
type Extra struct {
	LpFee       uint64 `json:"lpFee"`
	ProtocolFee uint64 `json:"protocolFee"`

	// RateNumerator / RateDenominator are the values FPMM._getRateFeed returns:
	// SortedOracles.medianRate scaled down by 1e6, already swapped when the
	// pool's invertRateFeed flag is set. Price of token0 in token1 = num/den.
	RateNumerator   *uint256.Int `json:"rateNum"`
	RateDenominator *uint256.Int `json:"rateDen"`
	// RateTimestamp is SortedOracles.medianTimestamp; RateExpiry is
	// getTokenReportExpirySeconds. The rate is valid while
	// RateTimestamp + RateExpiry >= now.
	RateTimestamp uint64 `json:"rateTs"`
	RateExpiry    uint64 `json:"rateExpiry"`

	// TradingMode is BreakerBox.getRateFeedTradingMode; only 0 allows swaps.
	TradingMode uint8 `json:"tradingMode"`
	// EnforceMarketHours is true when the pool's MarketHoursBreaker applies
	// the FX calendar (see closedMarketProbeTimestamp).
	EnforceMarketHours bool `json:"marketHours"`
	// Unquoteable is set when a read the swap path depends on reverted (for
	// example the rate feed is not registered in the BreakerBox).
	Unquoteable bool `json:"unquoteable,omitempty"`

	Limits [2]TradingLimit `json:"limits"`
}

// SwapInfo carries the post-swap trading-limit state that UpdateBalance
// applies, so it is never recomputed.
type SwapInfo struct {
	Limits [2]TradingLimit `json:"limits"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
}
