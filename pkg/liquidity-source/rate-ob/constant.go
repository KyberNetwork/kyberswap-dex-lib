package rateob

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType = "rate-ob"

	// Price levels read per side. A taker sweep matches at most MatchingEngine.maxMatches orders
	// (20 on every live deployment), so 20 levels cover any single swap.
	bookDepth = 20
	// Orders read per level: maxMatches fills plus the evictions one matchAt pass may make.
	ordersPerLevel = 28
	// MatchingLib.MAX_DUST_EVICTIONS: unfillable orders one matchAt pass removes before it yields.
	maxDustEvictions = 8
	pairBatchSize    = 100

	// Measured on an anvil fork of Base mainnet (8453) against the deployed engine: limitSell /
	// limitBuy transactions with 1, 3 and 10 matches used ~362-365k, ~546k and ~1,186-1,306k gas,
	// the top of each range being a WETH leg paid out as native coin. Fitted as gasBase +
	// gasMatch*matches with the 21k intrinsic cost removed, rounded up so every measured run is
	// covered. (The earlier Robinhood Chain Testnet engine measured ~10% lower.)
	gasBase  = 250000
	gasMatch = 104000
	// Evicting a dust or expired order is a delete, a refund transfer and an event. No eviction
	// occurred in the measured runs, so this is an estimate.
	gasEviction = 45000
	// The band pool leg, fitted on the same Base fork the same way (21k removed, rounded up to
	// cover every measured run): what a remainder costs on top of the book walk -- the stop-order
	// handoff moving it out and back with an empty activation scan, PoolFallbackLib.route
	// (getPool, convert, band 0's fee, approve, two balance reads, the price report) and
	// BandPool.swap (the TWAP lookup, both transfers) -- plus each band filled. Measured whole
	// transactions: 11 matches + 1 band ~1,669k, 15 + 2 ~2,107k, 15 + 3 ~2,144k (buys, the WETH
	// leg paid as native coin), 20 + 2 ~2,301k and 20 + 3 ~2,343k (sells); pool-only 1-3 bands
	// ~391-493k. Pool-only swaps come in ~100k under the fit, since gasBase prices a match they
	// do not make.
	gasPoolLeg  = 215000
	gasPoolBand = 40000

	// BandPool / PoolBands constants.
	maxBands   = 8         // PoolBands.MAX_BANDS
	maxFeeRate = 3_000_000 // PoolBands.MAX_FEE_RATE: no band charges more than 3%
	// anchorWindow is BandPool.TWAP_WINDOW: the anchor is Orderbook.twap(300).
	anchorWindow = 300
)

// denom is MatchingEngine.DENOM: prices are quote per base x 1e8 and fees are numerators over it.
var denom = uint256.NewInt(1e8)

// minReportDivisor is BandPool.MIN_REPORT_DIVISOR: a band fill reports its price only when it
// pays out at least 1/1,000,000 of the band's payable reserve.
var minReportDivisor = uint256.NewInt(1_000_000)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidExtra          = errors.New("invalid extra")
	ErrZeroAmountOut         = errors.Join(pool.ErrSwapAmountBelowMin, errors.New("zero amount out"))
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrOrderSizeTooSmall     = errors.New("order size too small")
	ErrOverflow              = errors.New("overflow")
	ErrCallFailed            = errors.New("state call failed")
)
