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

	// defaultGas is the measured cost of MentoV3Adapter.executeMentoV3 (quote,
	// transfer in, FPMM.swap) on the Monad USDC/USDm pool: 263,582 gas in the
	// ks-dex-adapter-lib fork test, both directions within 10k of each other.
	defaultGas int64 = 265_000

	// bps is FPMM.BASIS_POINTS_DENOMINATOR.
	bps = 10_000

	// tradingModeBidirectional is IBreakerBox's only trading mode that permits
	// swaps; OracleAdapter.getFXRateIfValid reverts for any other value.
	tradingModeBidirectional = 0

	// TradingLimitsV2 windows and precision.
	limitWindow0          = 300   // TIMESTEP0, 5 minutes
	limitWindow1          = 86400 // TIMESTEP1, 1 day
	limitInternalDecimals = 15    // INTERNAL_DECIMALS

	// rateStalenessBufferSeconds is subtracted from the oracle report expiry so
	// a quote issued right before the rate expires is not routed and then
	// reverted with NoRecentRate() at execution.
	rateStalenessBufferSeconds = 30

	// closedMarketProbeTimestamp is Saturday 2024-01-06 00:00:00 UTC. The
	// tracker asks the pool's MarketHoursBreaker whether the FX market is open
	// at this instant: a breaker that enforces FX hours answers false, the
	// always-open breaker used by stablecoin pools answers true. That tells the
	// simulator whether to apply the market-hours calendar off-chain.
	closedMarketProbeTimestamp = 1_704_499_200

	// sortedOraclesDenominator is the fixidity scale SortedOracles.medianRate
	// always returns as denominator; OracleAdapter asserts it.
	sortedOraclesDenominatorStr = "1000000000000000000000000"
	// oracleAdapterScaleDown is the 1e6 OracleAdapter divides both rate terms by.
	oracleAdapterScaleDownStr = "1000000"
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
