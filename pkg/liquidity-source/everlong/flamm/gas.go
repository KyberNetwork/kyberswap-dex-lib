package everlongflamm

import (
	"math"
	"math/big"
	"slices"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// A fill's gas is charged per fill, never above what the fill spends inside the pool's `swap` call net of the refund
// it earns: the least measured gas of the paths the fill may take, given what the refresh read at its block. A path
// is the venue calls a fill makes, as bits: what pays the output past custody, and whether the input repays a debt
// first. A fill paid from custody with no repay has none.
const (
	// pathOut: the output leaves a venue, as a withdrawal of supplied output, a release of posted output or a borrow.
	pathOut uint8 = 1 << iota
	// pathFirst: the output is borrowed where it owed nothing before the fill.
	pathFirst
	// pathOwing: the output is borrowed where it owed already.
	pathOwing
	// pathRepay: the input repays a debt before the payout.
	pathRepay
)

// A token's debt, as the pool's request reads it (its legs): not read, none, or some.
const (
	owedUnknown uint8 = iota
	owedNone
	owedSome
)

// freedSlack: what the pool frees before a draw grows with its venues' supply interest between the refresh block
// and the fill, so a payout past it by up to 1/freedSlack of it may still be paid with no borrow.
const freedSlack = 10_000

// canRepay is the `venueActions` bit of a venue that takes a repay now (IFinancingAccount.CAN_REPAY).
const canRepay uint8 = 2

// chainKinds are the Router's venue kinds each chain routes: those with cells in the table whose guards this
// source builds. A pair with a live venue of any other kind is not routed. Lista Moolah (3) holds a minimum
// position, which its ladders' guard keeps out of the window it opens (guard.go).
var chainKinds = map[valueobject.ChainID][]uint8{
	valueobject.ChainIDBase:     {0, 1},
	valueobject.ChainIDEthereum: {0, 1},
	valueobject.ChainIDBSC:      {1, 2, 3},
}

// Cell is one path a fill of a direction may take, with the gas a fill on it is charged.
type Cell struct {
	Path uint8 `json:"p"`
	Gas  int64 `json:"g"`
}

// gasCell is one cell of the generated table (gas_table.go): the hook build (HookFamily.CodeHash), the chain, the
// Router's venue kind, the side (sale: token 0 in) and the path, with the gas a fill on that path is charged, the
// least measured of its fills less 1 %.
type gasCell struct {
	family string
	chain  valueobject.ChainID
	kind   uint8
	sale   bool
	path   uint8
	gas    int64
}

// cellsOf are the table's cells of a hook build, a chain, a venue kind and a side.
func cellsOf(family string, chain valueobject.ChainID, kind uint8, sale bool) []Cell {
	var cs []Cell
	for _, c := range gasCells {
		if strings.EqualFold(c.family, family) && c.chain == chain && c.kind == kind && c.sale == sale {
			cs = append(cs, Cell{Path: c.path, Gas: c.gas})
		}
	}
	return cs
}

// gasSecondFill is one second-fill charge of the generated table (gas_table.go): the hook build, the chain, the
// Router's venue kind and the side, with the gas a fill on a pool another fill of the route moved is charged, the
// least measured of the second of two swaps of one pool in one transaction less 1 %.
type gasSecondFill struct {
	family string
	chain  valueobject.ChainID
	kind   uint8
	sale   bool
	gas    int64
}

// secondOf is the table's second-fill charge of a hook build, a chain, a venue kind and a side; false: none measured.
func secondOf(family string, chain valueobject.ChainID, kind uint8, sale bool) (int64, bool) {
	for _, s := range gasSecondFills {
		if strings.EqualFold(s.family, family) && s.chain == chain && s.kind == kind && s.sale == sale {
			return s.gas, true
		}
	}
	return 0, false
}

// afterOf are the table's cells of a hook build, a chain, a venue kind and a side for a pool's first fill after
// another pool of this source filled earlier in the same transaction (gasCellsAfter): every path of cellsOf, at its
// measured gas after another pool where the table has one, never above its cell, and at its cell where it has none.
func afterOf(family string, chain valueobject.ChainID, kind uint8, sale bool) []Cell {
	cs := cellsOf(family, chain, kind, sale)
	for _, c := range gasCellsAfter {
		if !strings.EqualFold(c.family, family) || c.chain != chain || c.kind != kind || c.sale != sale {
			continue
		}
		if at := slices.IndexFunc(cs, func(e Cell) bool { return e.Path == c.path }); at >= 0 && c.gas > 0 {
			cs[at].Gas = min(cs[at].Gas, c.gas)
		}
	}
	return cs
}

// venueKinds are the kinds the config's chain routes, narrowed to the config's VenueKinds where it names any.
func (c *Config) venueKinds() []uint8 {
	all := chainKinds[c.ChainID]
	if len(c.VenueKinds) == 0 {
		return all
	}
	return slices.DeleteFunc(slices.Clone(all), func(k uint8) bool { return !slices.Contains(c.VenueKinds, k) })
}

// chargeReads are the venue reads of a direction's charge and guard: the cells of the pair's live kinds, each path
// at its least, as a fill may draw on any of them, and the least of their second-fill charges; the same paths
// at their least after another pool of the source filled (afterOf), or none where that is the cells themselves;
// whether a borrow of the output may be a first one while it owes, where more than one live venue lends it;
// whether a repay of the input may be certain where it owes, as every venue lending it is live and takes repays now;
// and the live venue holding a minimum position that lends the input or the output, where one does.
type chargeReads struct {
	cells, after        []Cell
	second              int64
	firstOwing, repayOk bool
	heldIn, heldOut     *heldRepay
}

// charges is the kind gate and the venue reads of the charge for a fill of token i for token j by a pricer of build
// `family`. The pair is routed only where every venue record read, some live venue's credit pair holds i or j, and
// every such venue is of a kind the chain routes with cells and a second-fill charge for the build and the side.
func (t *PoolTracker) charges(st *poolState, family string, i, j int) (chargeReads, bool) {
	var r chargeReads
	if st.venueUnread || family == "" {
		return r, false
	}
	kinds := t.cfg.venueKinds()
	var live [256]bool
	var lendIn, lendOut int
	r.repayOk = i >= maxTokens || st.idleDebt>>i&1 == 0
	for _, v := range st.venues {
		c, d := int(v.Coll), int(v.Debt)
		if c != i && c != j && d != i && d != j {
			continue
		}
		if !slices.Contains(kinds, v.Kind) {
			return r, false
		}
		live[v.Kind] = true
		if d == j {
			lendOut++
			if v.held {
				r.heldOut = &heldRepay{floor: v.floor}
			}
		}
		if d == i {
			lendIn++
			if v.held {
				r.heldIn = &heldRepay{floor: v.floor}
			}
			if !v.actionsOk || v.actions&canRepay == 0 {
				r.repayOk = false
			}
		}
	}
	for k := range live {
		if !live[k] {
			continue
		}
		cs := cellsOf(family, t.cfg.ChainID, uint8(k), i == 0)
		second, ok := secondOf(family, t.cfg.ChainID, uint8(k), i == 0)
		if len(cs) == 0 || !ok || second <= 0 {
			return r, false
		}
		if r.second == 0 || second < r.second {
			r.second = second
		}
		r.cells = leastByPath(r.cells, cs)
		r.after = leastByPath(r.after, afterOf(family, t.cfg.ChainID, uint8(k), i == 0))
	}
	if len(r.cells) == 0 {
		return r, false // no live venue on the pair: no fill of such a pool was measured
	}
	byPath := func(a, b Cell) int { return int(a.Path) - int(b.Path) }
	slices.SortFunc(r.cells, byPath)
	slices.SortFunc(r.after, byPath)
	if slices.Equal(r.after, r.cells) {
		r.after = nil // no fill after another pool measured on any path: the ladder carries the cells alone
	}
	r.firstOwing = lendOut > 1
	if r.heldOut != nil {
		r.heldOut.sole = lendOut == 1
	}
	if r.heldIn != nil {
		r.heldIn.sole = lendIn == 1
	}
	return r, true
}

// leastByPath joins a kind's cells to those of the kinds before it, each path at its least.
func leastByPath(to, cs []Cell) []Cell {
	for _, c := range cs {
		if at := slices.IndexFunc(to, func(e Cell) bool { return e.Path == c.Path }); at < 0 {
			to = append(to, c)
		} else {
			to[at].Gas = min(to[at].Gas, c.Gas)
		}
	}
	return to
}

// readLegs sets the charge's reads from the pool's request for this direction (legs: heldIn, suppliedIn, postedIn,
// owedIn, then the same of the output), where they read: the band word is not zero. Freed is what the pool frees
// of the output before a draw, its held, supplied and posted output. PostedIn is whether the input has collateral
// posted. A repay is certain where the input owes and the venues lending it take repays now (repayOk), and, where
// one of them holds a minimum position (heldIn), where it is the only one and the debt is past its minimum by the
// margin (heldRepayOk), as it takes nothing of a partial repay that would leave less.
func (l *Ladder) readLegs(w pairWords, repayOk bool, heldIn *heldRepay) {
	if !legsRead(w) {
		return
	}
	freed := new(big.Int).Add(w.legs[4], w.legs[5])
	freed.Add(freed, w.legs[6])
	if f, overflow := uint256.FromBig(freed); !overflow {
		l.Freed = f
	}
	l.OwedOut, l.OwedIn = owedOf(w.legs[7]), owedOf(w.legs[3])
	l.PostedIn = w.legs[2].Sign() > 0
	l.Repay = l.OwedIn == owedSome && repayOk
	if l.Repay && heldIn != nil {
		owed, overflowO := uint256.FromBig(w.legs[3])
		var floor *uint256.Int
		overflowF := true
		if heldIn.floor != nil {
			floor, overflowF = uint256.FromBig(heldIn.floor)
		}
		l.Repay = heldIn.sole && !overflowO && !overflowF && heldRepayOk(owed, floor)
	}
}

// legsRead: the request's words read, its band word not zero.
func legsRead(w pairWords) bool {
	if w.band == nil || w.band.Sign() == 0 {
		return false
	}
	return !slices.Contains(w.legs[:], nil)
}

// swapLegs are a direction's request words seen from the other direction of the pair: its input's legs are the
// other's output's.
func swapLegs(w pairWords) pairWords {
	s := w
	copy(s.legs[:4], w.legs[4:])
	copy(s.legs[4:], w.legs[:4])
	return s
}

func owedOf(owed *big.Int) uint8 {
	if owed.Sign() == 0 {
		return owedNone
	}
	return owedSome
}

// heldRepayOk: a repay at a venue that holds a minimum position (IFinancingAccount.CAP_HELD_REPAY) is certain only
// where the debt is past its floor (IHeldRepay.minimumDebt) by a margin, floor x 1.01 + 4, as the venue takes
// nothing of a partial repay at or under its minimum and the minimum moves with its oracle after the read.
func heldRepayOk(owed, floor *uint256.Int) bool {
	var bar, d uint256.Int
	bar.Add(floor, d.SetUint64(99))
	bar.Div(&bar, d.SetUint64(100)) // ceil(floor / 100)
	if _, overflow := bar.AddOverflow(&bar, floor); overflow {
		return false
	}
	if _, overflow := bar.AddOverflow(&bar, d.SetUint64(4)); overflow {
		return false
	}
	return owed.Gt(&bar)
}

// charge is the gas of a fill paying `q` from the direction's start: the least of the cells the reads allow, or
// the least of all where the reads allow none. Where the pool moved since the refresh, the fill is another of
// the same route, in the same transaction, on what the fill before it warmed: it is charged the second-fill charge,
// which the reads, moved by that fill, do not narrow. Where it has not moved and another pool of the source
// filled earlier in the route (after), the fill is the pool's first on what that pool warmed of the contracts and
// storage the two share: the same paths are charged their gas measured after another pool (the ladder's After
// cells), never above their cells. The second-fill charge stands either way: the pool's own fill before it warmed
// all another pool's would.
func (c *curve) charge(q *uint256.Int, moved, after bool) int64 {
	if moved {
		return c.second
	}
	cells, least := c.cells, c.least
	if after {
		cells, least = c.after, c.leastAfter
	}
	g := int64(math.MaxInt64)
	for _, cell := range cells {
		if cell.Gas < g && c.allows(cell.Path, q) {
			g = cell.Gas
		}
	}
	if g == math.MaxInt64 {
		return least
	}
	return g
}

// allows reports whether a fill paying q may take the path, by what the refresh read. Past the output's
// custody a venue pays, and past what the pool frees before a draw a borrow does. A borrow is a first one where
// the output owes nothing, an owing one where it owes, and either where a second venue lends it. Where the output
// owes nothing and the input has collateral posted, it is either too: the pool posts only what a borrow needs past
// what is posted, often nothing, unlike every first borrow measured, so such a borrow may cost less than those, and
// it costs no less than an owing one. No repay where the input owes nothing, and a repay where it owes and the
// repay is certain. A supply of the input is never assumed, so no path is refused for lacking one.
func (c *curve) allows(path uint8, q *uint256.Int) bool {
	if c.heldRead && q.Gt(&c.held) && path&pathOut == 0 {
		return false
	}
	if c.freedRead && q.Gt(&c.freedTop) && path&(pathFirst|pathOwing) == 0 {
		return false
	}
	if path&(pathFirst|pathOwing) != 0 {
		if c.owedOut == owedNone && path&pathFirst == 0 && !c.postedIn {
			return false
		}
		if c.owedOut == owedSome && path&pathOwing == 0 && !c.firstOwing {
			return false
		}
	}
	if c.owedIn == owedNone && path&pathRepay != 0 {
		return false
	}
	return !c.repay || path&pathRepay != 0
}
