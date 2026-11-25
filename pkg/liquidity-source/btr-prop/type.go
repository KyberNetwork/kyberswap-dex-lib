package btrprop

import (
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/holiman/uint256"
)

var (
	// ErrBTRGated is returned when a leg is halted or swapped off.
	ErrBTRGated = errors.New("btr: leg gated or halted")
	// ErrBTRGatedCaller is returned when SWAP_GATED is set and the executor is not in
	// the pool's swap gate: every swap reverts NotAuthorized.
	ErrBTRGatedCaller = errors.New("btr: pool is SWAP_GATED; executor not in the swap gate")
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
	// ErrBTRStale is returned when the sampled ladders are older than MaxAge at quote time.
	ErrBTRStale = errors.New("btr: sampled state older than MaxAge")
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
	// GateOK is true when Pool.inSwapGate(executor) holds, which opens SWAP_GATED
	// legs to the executor. False when unreadable.
	GateOK bool `json:"k,omitempty"`
	// Ladders[from][to] is the curve for Tokens[from] -> Tokens[to]; the diagonal
	// and pairs without a sample are empty.
	Ladders [][]Ladder `json:"l"`
	// Hops[from][to] is the number of nodes on the pool's route for the pair (2 = a direct edge
	// through the hub, more = a spoke-to-spoke route through the anchor tree); zero when unknown.
	Hops [][]uint8 `json:"h,omitempty"`
	// Time is the block timestamp the ladders were sampled at. Zero for a state written before
	// the haircut moved to quote time: its ladders already carry the fixed-window haircut.
	Time uint64 `json:"t,omitempty"`
	// Stale[i] replays token i's staleness premium (zero when its word was unreadable).
	Stale []StaleLeg `json:"s,omitempty"`
	// Cuts[from][to] sizes the pair's haircut at quote time; empty without a ladder.
	Cuts [][]PairCut `json:"c,omitempty"`
	// Cash[i] is the most of token i the pool pays from liquid cash alone (liquidRoom); an output
	// past it makes the pool recall from the leg's hook inside the swap, which costs RecallGas[i]
	// more. Both are empty when no leg has a usable recall.
	Cash      []uint256.Int `json:"q,omitempty"`
	RecallGas []int64       `json:"r,omitempty"`
}

// setRecall records, per leg with usable recall, its liquid-only room and recall gas.
func (e *Extra) setRecall(n int, recalls []recall, liquids []*big.Int, assets []*rpcAssetResult) {
	for i, r := range recalls {
		if r.Gas == 0 || r.Room.IsZero() || assets[i] == nil {
			continue
		}
		cash := liquidRoom(liquids[i], &assets[i].Asset)
		if cash == nil {
			continue
		}
		if e.Cash == nil {
			e.Cash, e.RecallGas = make([]uint256.Int, n), make([]int64, n)
		}
		e.Cash[i], e.RecallGas[i] = *cash, r.Gas
	}
}

// swapGas is gas plus the recall gas of the output leg when taken (earlier swaps of the route's
// quotes) plus out reaches past its liquid cash.
func (e *Extra) swapGas(from, to int, taken, out *uint256.Int) int64 {
	g := e.gas(from, to)
	if to < 0 || to >= len(e.RecallGas) || e.RecallGas[to] == 0 {
		return g
	}
	var need uint256.Int
	if _, over := need.AddOverflow(taken, out); over || need.Gt(&e.Cash[to]) {
		g += e.RecallGas[to]
	}
	return g
}

// setCut records the haircut inputs of from -> to in an n x n matrix.
func (e *Extra) setCut(n, from, to int, c PairCut) {
	if e.Cuts == nil {
		e.Cuts = make([][]PairCut, n)
		for i := range e.Cuts {
			e.Cuts[i] = make([]PairCut, n)
		}
	}
	e.Cuts[from][to] = c
}

// factor returns num/den, the share of the sampled ladder of from -> to that a swap quoted age
// seconds after the sample may promise: the path's premium growth over age + InclusionAllowance.
// An Extra without a sample time quotes its ladders as stored. ok is false without haircut data.
func (e *Extra) factor(from, to int, age uint64) (num, den uint64, ok bool) {
	if e.Time == 0 {
		return 1, 1, true
	}
	if from < 0 || from >= len(e.Cuts) || to < 0 || to >= len(e.Cuts[from]) {
		return 0, 0, false
	}
	return e.Cuts[from][to].factor(e.Stale, e.Time, age+uint64(InclusionAllowance/time.Second))
}

// setHops records the route length of from -> to in an n x n matrix.
func (e *Extra) setHops(n, from, to, hops int) {
	if e.Hops == nil {
		e.Hops = make([][]uint8, n)
		for i := range e.Hops {
			e.Hops[i] = make([]uint8, n)
		}
	}
	e.Hops[from][to] = uint8(min(hops, math.MaxUint8))
}

// gas is the swap_qe gas estimate for from -> to: DefaultGas for a direct edge, more for every
// extra node the route crosses.
func (e *Extra) gas(from, to int) int64 {
	if from < 0 || from >= len(e.Hops) || to < 0 || to >= len(e.Hops[from]) {
		return DefaultGas
	}
	return DefaultGas + int64(max(int(e.Hops[from][to])-2, 0))*GasPerExtraHop
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
	// swap disabled, or SWAP_GATED with the executor outside the swap gate).
	Gated bool `json:"gated"`
}
