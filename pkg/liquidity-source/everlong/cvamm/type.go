package everlongcvamm

import (
	"math/big"

	"github.com/holiman/uint256"
)

// StaticExtra is the immutable per-venue metadata written at listing time. The pool
// address IS the ALM address, so only the optional periphery and gas overrides live here.
type StaticExtra struct {
	Adapter       string `json:"adapter,omitempty"`
	Quoter        string `json:"quoter,omitempty"`
	GasStableIn   int64  `json:"gasSIn,omitempty"`
	GasVolatileIn int64  `json:"gasVIn,omitempty"`
}

// Support is the funded band from getSupport(): anchor-free, invalidated only by a
// CurveRetuned (tracked implicitly — every refresh re-reads it). xLo is the high-price
// edge (volatile leg exactly zero there), xHi the low-price edge (stable leg exactly
// zero), yHi = y(xHi), the stable offset.
type Support struct {
	AWad *uint256.Int `json:"a"`
	XLo  *uint256.Int `json:"xLo"`
	XHi  *uint256.Int `json:"xHi"`
	YHi  *uint256.Int `json:"yHi"`
}

// Extra is the per-refresh venue state — everything a quote needs, all read from the
// ALM pinned at one block. XWad is the AUTHORITATIVE inventory coordinate (the price is
// a lossy floored sqrt of it and is deliberately not stored). Reserves are the accounted
// tradeable reserves (idle excluded) — authoritative for the solvency clamp. The
// directional fees are READ, never computed: their realized-variance input has no getter
// and no event on-chain.
type Extra struct {
	Support          Support      `json:"sup"`
	XWad             *uint256.Int `json:"x"`
	AnchorSqrtX96    *uint256.Int `json:"anchor"`
	Kappa            *uint256.Int `json:"kappa"`
	FeeStableInWad   *uint256.Int `json:"feeS"`
	FeeVolatileInWad *uint256.Int `json:"feeV"`
	Paused           bool         `json:"paused,omitempty"`
}

// supportRaw is the ethrpc decode target for getSupport() (tuple field order = ABI order).
type supportRaw struct {
	AWad *big.Int
	XLo  *big.Int
	XHi  *big.Int
	YHi  *big.Int
}

// PoolMeta carries what the executor needs to build the direct
// CvammALM.swap(stableIn, amountIn, minAmountOut, sqrtPriceLimitX96, to, deadline) call:
// stableIn = (tokenIn == token0), sqrtPriceLimitX96 = 0 (no limit), and the ALM pulls the
// input under an allowance. Exact-input only; partial fills are NORMAL — read
// amountInUsed. Adapter (if set) is an ISwapRouter02.exactInputSingle shim that fails
// closed on partial fills.
type PoolMeta struct {
	ALM     string `json:"alm"`
	Adapter string `json:"adapter,omitempty"`
}

// SwapInfo carries the post-fill state from CalcAmountOut to UpdateBalance so the
// transition is never recomputed. GrossOut is the output-leg reserve decrease (net out +
// fee: the fee leaves the priced book into idle).
type SwapInfo struct {
	XAfter       *uint256.Int `json:"x"`
	AmountInUsed *uint256.Int `json:"in"`
	GrossOut     *uint256.Int `json:"out"`
	StableIn     bool         `json:"sIn"`
}
