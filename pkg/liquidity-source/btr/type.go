package btr

import "errors"

var (
	// ErrBTRGated is returned when a leg is halted, swapped off, or otherwise
	// not on the open path. The aggregator treats it as "no route".
	ErrBTRGated = errors.New("btr: leg gated or halted")
	// ErrBTRGatedLane is returned when SWAP_GATED is set and the executor does
	// not hold lane 0x400. The launch Monad core is gated; every route reverts
	// NotAuthorized until BTR governance clears it or grants a lane.
	ErrBTRGatedLane = errors.New("btr: pool is SWAP_GATED; executor needs lane 0x400")
	// ErrBTRNoRoute is returned when no sampled ladder exists for the pair.
	ErrBTRNoRoute = errors.New("btr: no sampled ladder for pair")
	// ErrBTRNoFill is returned when a sampled quote yields zero.
	ErrBTRNoFill = errors.New("btr: no fill at this size")
	// ErrBTRAmountTooLarge is returned above the largest sampled amount; the
	// ladder is never extrapolated (fail closed).
	ErrBTRAmountTooLarge = errors.New("btr: amount above sampled range")
	// ErrBTRInvalidAmount is returned for a nil/non-positive amount in.
	ErrBTRInvalidAmount = errors.New("btr: invalid amount in")
	// ErrBTRBadState is returned when serialized state cannot be trusted.
	ErrBTRBadState = errors.New("btr: invalid pool state")
)

// DexType is the dex-lib pool type for BTR's AIMM book.
const DexType = "btr"

// StaticExtra holds per-pool constants that never change.
type StaticExtra struct {
	Pool      string   `json:"pool"`
	Factory   string   `json:"factory"`
	Tokens    []string `json:"tokens"`
	BaseToken string   `json:"baseToken"`
	// Router is recorded for reference only; see RouterAddress. The executor
	// must call Pool.swap_qe directly, not the Router.
	Router string `json:"router"`
}

// Point is one on-chain-probed sample of the curve for an ordered pair:
// quoting AmountIn (a decimal string, exact wei) returns AmountOut. Amounts are
// strings, not float64: the aggregator's floor is wei-exact and the differential
// test compares at 1 bps, so the cached samples must not lose low-order bits.
type Point struct {
	In  string `json:"in"`
	Out string `json:"out"`
}

// Extra is the mutable per-block state. BTR prices from keeper-pushed marks in
// a separate store, so a reserve-only model is wrong: the tracker probes the
// on-chain `getSwapQuote` view over a grid per ordered pair and stores the
// resulting ladder. `getSwapQuote` is side-effect-free and omits BTR's
// execution guards, so flags are cached too and every leg is gated before a
// ladder is used (M1/M5). The router's min-out floor still comes from a pinned
// `swap_qe` simulation in the on-chain adapter, never from this ladder.
type Extra struct {
	BlockNumber uint64 `json:"blockNumber"`
	// Flags[token] is the raw 16-bit word from Pool.getRiskFlags(token).
	Flags map[string]uint16 `json:"flags"`
	// Ladders["in|out"] is the sampled curve, ascending by In, for that
	// ordered pair (lower-cased addresses).
	Ladders map[string][]Point `json:"ladders"`
}

// PoolMeta is the GetMetaInfo payload. ApprovalAddress is the core pool: the
// executor approves the pool and swap_qe pulls amountIn via transferFrom.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
	// Gated reports whether every cached leg is on the open path. While the
	// live core is SWAP_GATED this is false and CalcAmountOut returns no route.
	Gated bool `json:"gated"`
}

// SwapInfo is the execution hint for the on-chain adapter. The adapter encodes
// the same fields as calldata (`ks-dex-adapter-lib`); this is informational for
// the route builder and is not the min-out guarantee.
type SwapInfo struct {
	Pool      string `json:"pool"`
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	AmountIn  string `json:"amountIn"`
	MinOut    string `json:"minOut"`
	Recipient string `json:"recipient"`
	Deadline  string `json:"deadline"`
}

// pairKey is the canonical map key for an ordered pair.
func pairKey(tokenIn, tokenOut string) string {
	return lowerAddr(tokenIn) + "|" + lowerAddr(tokenOut)
}
