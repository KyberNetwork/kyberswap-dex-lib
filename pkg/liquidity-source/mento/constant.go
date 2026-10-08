package mento

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeMento

	// FPMMFactory
	methodDeployedFPMMAddresses = "deployedFPMMAddresses"

	// FPMM
	methodMetadata            = "metadata"
	methodGetReserves         = "getReserves"
	methodLpFee               = "lpFee"
	methodProtocolFee         = "protocolFee"
	methodOracleAdapter       = "oracleAdapter"
	methodReferenceRateFeedID = "referenceRateFeedID"
	methodInvertRateFeed      = "invertRateFeed"
	methodGetTradingLimits    = "getTradingLimits"
	methodGetAmountOut        = "getAmountOut"

	// OracleAdapter
	methodSortedOracles      = "sortedOracles"
	methodBreakerBox         = "breakerBox"
	methodMarketHoursBreaker = "marketHoursBreaker"

	// SortedOracles
	methodMedianRate                  = "medianRate"
	methodMedianTimestamp             = "medianTimestamp"
	methodGetTokenReportExpirySeconds = "getTokenReportExpirySeconds"

	// BreakerBox
	methodGetRateFeedTradingMode = "getRateFeedTradingMode"

	// MarketHoursBreaker
	methodIsFXMarketOpen = "isFXMarketOpen"

	// defaultGas is executeSolidlyV2 (transfer in, getAmountOut, FPMM.swap) on the Monad
	// USDC/USDm pool in a Tenderly sim: 371,985 and 379,905 gas. Monad prices cold access
	// well above Ethereum, and the swap reads the oracle adapter, SortedOracles and BreakerBox.
	defaultGas int64 = 380_000

	// bps is FPMM.BASIS_POINTS_DENOMINATOR.
	bps = 10_000

	// tradingModeBidirectional is IBreakerBox's only trading mode that permits
	// swaps; OracleAdapter.getFXRateIfValid reverts for any other value.
	tradingModeBidirectional = 0

	// TradingLimitsV2 windows and precision.
	limitWindow0          = 300   // TIMESTEP0, 5 minutes
	limitWindow1          = 86400 // TIMESTEP1, 1 day
	limitInternalDecimals = 15    // INTERNAL_DECIMALS

	// executionDelaySeconds is the quote-to-execution allowance: the oracle rate
	// and FX market must stay valid this long, or the swap would revert on-chain.
	executionDelaySeconds = 30

	// closedMarketProbeTimestamp is Saturday 2024-01-06 00:00 UTC. A MarketHoursBreaker
	// that enforces FX hours reports closed here; the always-open one used by stablecoin
	// pools reports open. The tracker probes it to set Extra.EnforceMarketHours.
	closedMarketProbeTimestamp = 1_704_499_200
)

var (
	ErrInvalidToken          = errors.New("mento: token is not token0 or token1")
	ErrZeroAmountIn          = errors.New("mento: zero amount in")
	ErrZeroAmountOut         = errors.New("mento: zero amount out")
	ErrInsufficientLiquidity = errors.New("mento: amount out exceeds reserve")
	ErrUnquoteable           = errors.New("mento: pool state could not be read; oracle feed unavailable")
	ErrTradingSuspended      = errors.New("mento: breaker box has suspended trading on the rate feed")
	ErrNoRecentRate          = errors.New("mento: oracle rate is stale")
	ErrFXMarketClosed        = errors.New("mento: fx market is closed")
	ErrInvalidRate           = errors.New("mento: oracle rate has a zero term")
	ErrInvalidFee            = errors.New("mento: lp + protocol fee exceeds 100%")
	ErrL0LimitExceeded       = errors.New("mento: 5-minute trading limit exceeded")
	ErrL1LimitExceeded       = errors.New("mento: 1-day trading limit exceeded")
	ErrInt96Bounds           = errors.New("mento: trading limit netflow exceeds int96")
	ErrOverflow              = errors.New("mento: uint256 overflow")
)
