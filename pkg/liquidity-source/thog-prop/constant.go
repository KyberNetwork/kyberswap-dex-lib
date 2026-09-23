package thogprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeThogProp

	methodMakerSnapshot        = "makerSnapshot"
	methodMakerQuoteExactInput = "makerQuoteExactInput"

	// defaultGas is an estimate for makerSwapExactInput(); no verified source
	// exists to measure it precisely, so this mirrors the order of magnitude
	// of other single-call RFQ executors (parity-prop's defaultGas) until a
	// real Tenderly trace is available.
	defaultGas = 150000

	// spreadDenominator is exactQuote's SPREAD_DENOMINATOR -- 0.1 bps units,
	// 400000 == 100%.
	spreadDenominator = 400000

	// fastLaneFrictionBps is the AuctionHandler's documented worst-case
	// haircut applied when it's already warm: 9990/10000.
	fastLaneFrictionBps = 10

	exposureScale   = 1_000_000
	covarianceScale = 1_000_000_000
	lambdaScale     = 1_000_000

	// executionAgeLookahead is a deliberately small, fixed number of blocks
	// assumed to elapse between the tracker's snapshot and actual on-chain
	// execution. exactQuote's spread widens by perBlockWidening*age every
	// block, and maxAge (read live, never hardcoded) hard-reverts once
	// exceeded -- quoting age=0 would be optimistic and could promise output
	// the chain won't deliver by the time the tx lands.
	executionAgeLookahead = 2
)

var (
	ErrGloballyPaused     = errors.New("thog-prop: globally paused")
	ErrPriceSurfacePaused = errors.New("thog-prop: price surface paused")
	ErrPairRiskNotReady   = errors.New("thog-prop: pair risk state not ready")
	ErrRiskV3NotReady     = errors.New("thog-prop: XAUt0 risk v3 state not ready")
	ErrSideDisabled       = errors.New("thog-prop: sell or buy side disabled for token")
	ErrStalePrice         = errors.New("thog-prop: posted price stale or missing")
	ErrZeroMaxAge         = errors.New("thog-prop: maxAge is zero")
	ErrAmountTooLarge     = errors.New("thog-prop: amountIn exceeds input limit")
	ErrZeroNotional       = errors.New("thog-prop: zero notional")
	ErrSpreadTooWide      = errors.New("thog-prop: spread at/above denominator")
	ErrPenaltyExceedsOut  = errors.New("thog-prop: risk penalty exceeds gross output")
	ErrZeroAmountOut      = errors.New("thog-prop: zero amount out")
	ErrInsufficientBal    = errors.New("thog-prop: amount out exceeds pool balance")
	ErrSameToken          = errors.New("thog-prop: tokenIn equals tokenOut")
	ErrInvalidToken       = errors.New("thog-prop: unknown token")
)
