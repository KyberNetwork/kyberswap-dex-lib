package btrprop

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	// ErrBTRGated is returned when a leg is halted or swapped off.
	ErrBTRGated = errors.New("btr: leg gated or halted")
	// ErrBTRGatedLane is returned when SWAP_GATED is set and the executor does
	// not hold lane 0x400: every swap reverts NotAuthorized.
	ErrBTRGatedLane = errors.New("btr: pool is SWAP_GATED; executor needs lane 0x400")
	// ErrBTRNoRoute is returned when no sampled ladder exists for the pair.
	ErrBTRNoRoute = errors.New("btr: no sampled ladder for pair")
	// ErrBTRNoFill is returned when a sampled quote yields zero.
	ErrBTRNoFill = errors.New("btr: no fill at this size")
	// ErrBTRAmountTooLarge is returned above the largest sampled amount. The
	// ladder is never extrapolated: past the pool's liquidity cap swap_qe takes
	// the full input but pays only the capped output.
	ErrBTRAmountTooLarge = errors.New("btr: amount above sampled range")
	// ErrBTRInvalidAmount is returned for a nil/non-positive amount in.
	ErrBTRInvalidAmount = errors.New("btr: invalid amount in")
	// ErrBTRBadState is returned when the pool state cannot be trusted.
	ErrBTRBadState = errors.New("btr: invalid pool state")
)

// DexType is the dex-lib pool type for BTR's AIMM book.
const DexType = "btr-prop"

// Point is one on-chain-probed sample of the curve: quoting In returns Out.
type Point struct {
	In  uint256.Int
	Out uint256.Int
}

// Ladder is the sampled curve for one ordered pair, as [in, out] pairs ascending
// and strictly increasing in both. It marshals to bare arrays, as there are
// hundreds of samples per pool and keys would dominate the Extra size.
type Ladder [][2]uint256.Int

// newLadder packs pts into a non-nil Ladder, so a pair without samples marshals to [].
func newLadder(pts []Point) Ladder {
	l := make(Ladder, len(pts))
	for k := range pts {
		l[k] = [2]uint256.Int{pts[k].In, pts[k].Out}
	}
	return l
}

// points unpacks the samples into Points.
func (l Ladder) points() []Point {
	pts := make([]Point, len(l))
	for k := range l {
		pts[k].In, pts[k].Out = l[k][0], l[k][1]
	}
	return pts
}

// Extra is the mutable per-block state. BTR prices from keeper-pushed marks in
// a separate store, so a reserve-only model is wrong: the tracker probes the
// on-chain `getSwapQuote` view over a grid per ordered pair and stores the
// resulting ladder. The view omits BTR's execution guards, so the risk flags
// are cached too and every leg is gated before a ladder is used.
type Extra struct {
	// Flags[i] is the raw 16-bit word from Pool.getRiskFlags(Tokens[i]); 0 when
	// the read failed (treated as swap disabled).
	Flags []uint16 `json:"f"`
	// LaneOK is true when registry.perms(executor) includes lane 0x400, which
	// opens SWAP_GATED legs to the executor. False when unreadable.
	LaneOK bool `json:"k,omitempty"`
	// Ladders[from][to] is the curve for Tokens[from] -> Tokens[to]; the diagonal
	// and pairs without a sample are empty.
	Ladders [][]Ladder `json:"l"`
}

// ladder returns the points for Tokens[from] -> Tokens[to], or nil.
func (e *Extra) ladder(from, to int) []Point {
	if from < 0 || from >= len(e.Ladders) || to < 0 || to >= len(e.Ladders[from]) {
		return nil
	}
	return e.Ladders[from][to].points()
}

// PoolMeta is the GetMetaInfo payload. ApprovalAddress is the core pool: the
// executor approves the pool and swap_qe pulls amountIn via transferFrom.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
	// Gated reports whether any cached leg is closed to the executor (halted,
	// swap disabled, or SWAP_GATED without the executor's lane).
	Gated bool `json:"gated"`
}
