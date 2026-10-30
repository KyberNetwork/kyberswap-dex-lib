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

	// Measured on an anvil fork of Robinhood Chain Testnet: limitSell/limitBuy transactions with
	// 1, 2, 8 and 20 matches used ~336k, ~416k, ~887k and ~1,898k gas. Fitted as gasBase +
	// gasMatch*matches with the 21k intrinsic cost removed.
	gasBase  = 233000
	gasMatch = 82000
	// Evicting a dust or expired order is a delete, a refund transfer and an event. No eviction
	// occurred in the measured runs, so this is an estimate.
	gasEviction = 45000
)

// denom is MatchingEngine.DENOM: prices are quote per base x 1e8 and fees are numerators over it.
var denom = uint256.NewInt(1e8)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidExtra          = errors.New("invalid extra")
	ErrZeroAmountOut         = errors.Join(pool.ErrSwapAmountBelowMin, errors.New("zero amount out"))
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrOrderSizeTooSmall     = errors.New("order size too small")
	ErrOverflow              = errors.New("overflow")
	ErrCallFailed            = errors.New("state call failed")
)
