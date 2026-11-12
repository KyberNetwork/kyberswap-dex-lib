package everlongflamm

import (
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// quotesWithNoSwapLimit is quotesDigest's sha256 of the sweep's 3,604 quotes with no swap limit. They are the quotes
// the package gave before it read a limit, but for the sweep's quotes after a fill in the fill's own direction, which
// are refused since a direction takes one fill per route.
const quotesWithNoSwapLimit = "6d577940f89784acb10bec6c9fc9f0028a6e22a12e796657d8813f8bcd04e544"

// routeLimit is the swap limit of one request over these pools, as a router builds one: each simulator's
// CalculateLimit merged into one map, in an inventory of the exchange.
func routeLimit(sims ...pool.IPoolSimulator) pool.SwapLimit {
	limits := map[string]*big.Int{}
	for _, s := range sims {
		pool.MergeSwapLimits(limits, s)
	}
	return swaplimit.NewInventory(DexType, limits)
}

// limitPool is one of the BSC replay fixture's pools, BTCB/USDT on its venue kind, with the table's cells of its
// build: its recorded first refresh, its sale's ladder and its buy's.
type limitPool struct {
	name      string
	kind      uint8
	p         entity.Pool
	sale, buy Ladder
}

func limitPools(t *testing.T) []limitPool {
	t.Helper()
	fx := loadFixture(t, "testdata/bsc-fork.json.gz")
	kinds := map[string]uint8{"EverlongHook on Aave V3": 1, "EverlongHook on Venus Core": 2,
		"EverlongHook on Lista Moolah": 3}
	require.Len(t, fx.Pools, len(kinds))
	var ps []limitPool
	for _, p := range fx.Pools {
		kind, ok := kinds[fx.Kinds[p.Address]]
		require.True(t, ok, fx.Kinds[p.Address])
		lp := limitPool{name: fx.Kinds[p.Address], kind: kind, p: p}
		for _, l := range extraOf(t, p).Ladders {
			if l.In == 0 {
				lp.sale = l
			} else {
				lp.buy = l
			}
		}
		require.NotEmpty(t, lp.sale.Points, lp.name)
		require.NotEmpty(t, lp.buy.Points, lp.name)
		ps = append(ps, lp)
	}
	slices.SortFunc(ps, func(a, b limitPool) int { return int(a.kind) - int(b.kind) })
	return ps
}

func (lp limitPool) sim(t *testing.T) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: lp.p})
	require.NoError(t, err)
	return s
}

func (lp limitPool) ladders() []Ladder { return []Ladder{lp.sale, lp.buy} }

// at is a quote of the ladder's direction at its first point.
func (lp limitPool) at(l Ladder, limit pool.SwapLimit) pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: lp.p.Tokens[l.In].Address,
		Amount: l.Points[0][0].ToBig()}, TokenOut: lp.p.Tokens[l.Out].Address, Limit: limit}
}

func (lp limitPool) quote(t *testing.T, s *PoolSimulator, l Ladder, limit pool.SwapLimit) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := s.CalcAmountOut(lp.at(l, limit))
	require.NoError(t, err, lp.name)
	return res
}

// apply fills the ladder's direction at its first point, as a route does: the quote, then the balance update with
// the route's limit.
func (lp limitPool) apply(t *testing.T, s *PoolSimulator, l Ladder, limit pool.SwapLimit) *pool.CalcAmountOutResult {
	t.Helper()
	res := lp.quote(t, s, l, limit)
	s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: lp.at(l, limit).TokenAmountIn,
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo, SwapLimit: limit})
	return res
}

// afterGas is the table's charge after another pool for the path the table charges `gas` alone, on the pool's build,
// kind and the ladder's side: the same path's cell of gasCellsAfter.
func (lp limitPool) afterGas(t *testing.T, l Ladder, gas int64) int64 {
	t.Helper()
	cells := cellsOf(bscBuild, valueobject.ChainIDBSC, lp.kind, l.In == 0)
	after := afterOf(bscBuild, valueobject.ChainIDBSC, lp.kind, l.In == 0)
	require.Len(t, after, len(cells))
	at := slices.IndexFunc(cells, func(c Cell) bool { return c.Gas == gas })
	require.GreaterOrEqual(t, at, 0, "%s: %d is no cell of the key", lp.name, gas)
	require.Equal(t, 1, len(slices.DeleteFunc(slices.Clone(cells), func(c Cell) bool { return c.Gas != gas })),
		"one path at this gas")
	require.Equal(t, cells[at].Path, after[at].Path)
	return after[at].Gas
}

// sameButGas: two quotes of one fill that differ in nothing but the gas.
func sameButGas(t *testing.T, want, got *pool.CalcAmountOutResult, msg string) {
	t.Helper()
	w, g := *want, *got
	w.Gas, g.Gas = 0, 0
	require.Equal(t, w, g, msg)
}

// With no swap limit every quote is what it was before the package read one: the sweep's digest is the one taken on
// that code, though the BSC fixture's ladders now carry their cells after another pool. A limit no fill was applied
// with gives the same quotes, and so does one that lacks the count's key.
func TestWithNoSwapLimitEveryQuoteIsUnchanged(t *testing.T) {
	for _, lp := range limitPools(t) {
		for _, l := range lp.ladders() {
			require.Equal(t, afterOf(bscBuild, valueobject.ChainIDBSC, lp.kind, l.In == 0), l.After, lp.name)
			require.NotEqual(t, l.Cells, l.After, "%s: the ladder carries cells after another pool", lp.name)
		}
	}
	d, n := quotesDigest(t, func() pool.SwapLimit { return nil })
	require.Equal(t, 3604, n)
	require.Equal(t, quotesWithNoSwapLimit, d, "no limit")
	d, _ = quotesDigest(t, func() pool.SwapLimit {
		return swaplimit.NewInventory(DexType, map[string]*big.Int{limitFilled: new(big.Int)})
	})
	require.Equal(t, quotesWithNoSwapLimit, d, "a limit no fill was applied with")
	d, _ = quotesDigest(t, func() pool.SwapLimit { return swaplimit.NewInventory(DexType, map[string]*big.Int{}) })
	require.Equal(t, quotesWithNoSwapLimit, d, "a limit without the key")
}

// The source uses a swap limit, and every pool reports the count of the route's fills at zero, each its own integer.
func TestEveryPoolStartsTheRoutesCountAtZero(t *testing.T) {
	require.True(t, pool.UsesSwapLimit(DexType, DexType))
	ps := limitPools(t)
	a, b := ps[0].sim(t).CalculateLimit(), ps[1].sim(t).CalculateLimit()
	require.Equal(t, map[string]*big.Int{limitFilled: new(big.Int)}, a)
	a[limitFilled].SetInt64(7)
	require.Zero(t, b[limitFilled].Sign(), "no integer is shared")
	require.Zero(t, ps[0].sim(t).CalculateLimit()[limitFilled].Sign())
	limit := routeLimit(ps[0].sim(t), ps[1].sim(t), ps[2].sim(t))
	require.Zero(t, limit.GetLimit(limitFilled).Sign(), "the merged count")
	require.Equal(t, DexType, limit.GetExchange())
}

// A fill applied on one pool with the route's limit lowers the charge of another pool's first fill, and nothing
// else: that fill is charged the same path's cell after another pool, its amounts and swap info unchanged; the pool
// that filled is charged its second fill, as with no limit; a quote asked with no limit is charged as before; and
// once the other pool has filled too, its next fill is charged its second fill, as with no limit. Every pair of the
// fixture's pools, in either order, each side first and each side after: a split route's two legs on one side, and a
// route's two hops on either.
func TestAFillAfterAnotherPoolIsChargedItsCellsAfterAnotherPool(t *testing.T) {
	ps := limitPools(t)
	checked := 0
	for _, a := range ps {
		for _, b := range ps {
			if a.name == b.name {
				continue
			}
			for _, first := range a.ladders() {
				for _, next := range b.ladders() {
					sa, sb := a.sim(t), b.sim(t)
					limit := routeLimit(sa, sb)
					alone := b.quote(t, sb, next, nil)
					require.Equal(t, alone, b.quote(t, sb, next, limit), "no fill applied yet")
					firstAlone := a.quote(t, sa, first, nil)
					require.Equal(t, firstAlone, a.apply(t, sa, first, limit), "the route's first fill is charged its cells")
					require.Equal(t, int64(1), limit.GetLimit(limitFilled).Int64())

					after := b.quote(t, sb, next, limit)
					sameButGas(t, alone, after, a.name+" then "+b.name)
					want := b.afterGas(t, next, alone.Gas)
					require.Equal(t, want, after.Gas, "%s then %s: the path's cell after another pool", a.name, b.name)
					require.Less(t, after.Gas, alone.Gas)
					require.Equal(t, alone, b.quote(t, sb, next, nil), "asked with no limit: its cells")
					require.Equal(t, after, b.quote(t, sb, next, limit), "a quote changes nothing")
					require.Equal(t, int64(1), limit.GetLimit(limitFilled).Int64(), "a quote does not count")

					// the pool that filled: its second fill, the limit or not
					sa2 := a.sim(t)
					a.apply(t, sa2, first, nil)
					for _, l := range a.ladders() {
						moved, err := sa2.CalcAmountOut(a.at(l, nil))
						got, err2 := sa.CalcAmountOut(a.at(l, limit))
						require.Equal(t, err, err2)
						require.Equal(t, moved, got, "%s after its own fill", a.name)
					}

					// the other pool's own second fill, after another pool: its second fill, as with no limit
					b.apply(t, sb, next, limit)
					require.Equal(t, int64(2), limit.GetLimit(limitFilled).Int64())
					sb2 := b.sim(t)
					b.apply(t, sb2, next, nil)
					for _, l := range b.ladders() {
						moved, err := sb2.CalcAmountOut(b.at(l, nil))
						got, err2 := sb.CalcAmountOut(b.at(l, limit))
						require.Equal(t, err, err2)
						require.Equal(t, moved, got, "%s after another pool and its own fill", b.name)
						if err == nil {
							second, ok := secondOf(bscBuild, valueobject.ChainIDBSC, b.kind, l.In == 0)
							require.True(t, ok)
							require.Equal(t, second, got.Gas)
						}
					}
					checked++
				}
			}
		}
	}
	require.Equal(t, 24, checked)
}

// A split route through two of the source's pools, in either order: the leg applied first is charged its cells, the
// other its cells after another pool, so the two orders charge the route differently and each less than two first
// fills. A third pool after them is charged after another pool too.
func TestASplitRouteChargesTheLaterPoolAfterAnotherPoolInEitherOrder(t *testing.T) {
	ps := limitPools(t)
	for _, side := range []func(limitPool) Ladder{func(p limitPool) Ladder { return p.sale },
		func(p limitPool) Ladder { return p.buy }} {
		for _, order := range [][3]int{{0, 1, 2}, {1, 0, 2}, {2, 1, 0}, {0, 2, 1}} {
			sims := []*PoolSimulator{ps[0].sim(t), ps[1].sim(t), ps[2].sim(t)}
			limit := routeLimit(sims[0], sims[1], sims[2])
			var total, alone int64
			for k, at := range order {
				p, l := ps[at], side(ps[at])
				first := p.quote(t, sims[at], l, nil).Gas
				got := p.apply(t, sims[at], l, limit).Gas
				if k == 0 {
					require.Equal(t, first, got, "%s first", p.name)
				} else {
					require.Equal(t, p.afterGas(t, l, first), got, "%s after %d others", p.name, k)
				}
				total, alone = total+got, alone+first
			}
			var want int64
			for k, at := range order {
				p, l := ps[at], side(ps[at])
				g := p.quote(t, p.sim(t), l, nil).Gas
				if k > 0 {
					g = p.afterGas(t, l, g)
				}
				want += g
			}
			require.Equal(t, want, total)
			require.Less(t, total, alone)
		}
	}
}

// The count lives in the request's limit alone: a fill applied with one request's limit changes no quote of another
// request, on its own simulators or on the same ones, and nothing in the package remembers it.
func TestTheCountIsPerRequest(t *testing.T) {
	ps := limitPools(t)
	a, b, l := ps[0], ps[1], ps[1].sale
	sa1, sb1, sa2, sb2 := a.sim(t), b.sim(t), a.sim(t), b.sim(t)
	limit1, limit2 := routeLimit(sa1, sb1), routeLimit(sa2, sb2)
	alone := b.quote(t, sb1, l, nil)
	a.apply(t, sa1, a.sale, limit1)
	after := b.quote(t, sb1, l, limit1)
	require.Less(t, after.Gas, alone.Gas)
	require.Equal(t, alone, b.quote(t, sb2, l, limit2), "another request, its own simulators")
	require.Equal(t, alone, b.quote(t, sb1, l, limit2), "another request's limit on the same simulator")
	require.Zero(t, limit2.GetLimit(limitFilled).Sign())
	require.Equal(t, alone, b.quote(t, b.sim(t), l, routeLimit(a.sim(t), b.sim(t))), "a request after it")
	require.Equal(t, after, b.quote(t, sb2, l, limit1), "the first request's limit says so to any simulator")
}

// A cloned limit and the original do not see each other's fills, and a cloned pool leaves no mark of its own: a path
// explored on clones changes no quote of the route it was cloned from, and the other way round.
func TestACloneOfTheLimitIsIsolated(t *testing.T) {
	ps := limitPools(t)
	a, b, l := ps[2], ps[0], ps[0].buy
	sa, sb := a.sim(t), b.sim(t)
	limit := routeLimit(sa, sb)
	alone := b.quote(t, sb, l, limit)

	explored := limit.Clone()
	a.apply(t, sa.CloneState().(*PoolSimulator), a.buy, explored)
	require.Equal(t, int64(1), explored.GetLimit(limitFilled).Int64())
	require.Zero(t, limit.GetLimit(limitFilled).Sign(), "the original limit")
	require.Equal(t, alone, b.quote(t, sb, l, limit), "the route the path was cloned from")
	require.False(t, sa.moved(), "the original pool")
	after := b.quote(t, sb.CloneState().(*PoolSimulator), l, explored)
	require.Equal(t, b.afterGas(t, l, alone.Gas), after.Gas, "on the explored path")

	kept := limit.Clone()
	a.apply(t, sa, a.buy, limit)
	require.Zero(t, kept.GetLimit(limitFilled).Sign(), "a clone taken before the fill")
	require.Equal(t, alone, b.quote(t, sb, l, kept))
	require.Equal(t, after, b.quote(t, sb, l, limit))
	require.Equal(t, int64(1), limit.Clone().GetLimit(limitFilled).Int64(), "a clone taken after it carries the count")
}

// Only a fill that was applied counts: a balance update whose swap info is not this source's marks its direction but
// leaves the limit as it was, as does one of a direction the pool does not have; and a limit that lacks the key keeps no count, so every charge
// stays its cells.
func TestOnlyAnAppliedFillCounts(t *testing.T) {
	ps := limitPools(t)
	a, b := ps[0], ps[1]
	sa, sb := a.sim(t), b.sim(t)
	limit := routeLimit(sa, sb)
	res := a.quote(t, sa, a.sale, limit)
	in := a.at(a.sale, nil).TokenAmountIn
	sa.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee,
		SwapInfo: "another source's", SwapLimit: limit})
	sa.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: pool.TokenAmount{Token: in.Token,
		Amount: big.NewInt(1)}, Fee: *res.Fee, SwapInfo: res.SwapInfo, SwapLimit: limit})
	require.Zero(t, limit.GetLimit(limitFilled).Sign())
	require.True(t, sa.moved(), "a fill with unreadable swap info still marks its direction, and counts nowhere")

	sa = a.sim(t)
	bare := swaplimit.NewInventory(DexType, map[string]*big.Int{})
	alone := b.quote(t, sb, b.sale, nil)
	a.apply(t, sa, a.sale, bare)
	require.True(t, sa.moved(), "the fill itself is applied")
	require.Nil(t, bare.GetLimit(limitFilled))
	require.Equal(t, alone, b.quote(t, sb, b.sale, bare))

	a.apply(t, a.sim(t), a.sale, limit)
	require.Equal(t, int64(1), limit.GetLimit(limitFilled).Int64(), "an applied fill counts once")
}

// The curve's charge after another pool: each path at its After cell, never above its cell; an After cell of a path
// the ladder has no cell of is not read; where the reads allow no path, the least After cell; a fill after another on
// the pool keeps its second-fill charge; a ladder without After cells charges its cells; an After cell without gas is
// refused.
func TestCurveChargesTheCellsAfterAnotherPool(t *testing.T) {
	l := Ladder{Cells: []Cell{{0, 250_000}, {pathOut, 400_000}, {pathOut | pathFirst, 600_000}},
		After:  []Cell{{0, 180_000}, {pathOut, 450_000}, {pathOut | pathFirst, 520_000}, {pathRepay, 1_000}},
		Second: 100_000, Held: uint256.NewInt(100), Freed: uint256.NewInt(200), OwedOut: owedNone, OwedIn: owedNone}
	q := func(v uint64) *uint256.Int { return uint256.NewInt(v) }
	require.Equal(t, int64(250_000), chargeAt(t, l, q(100), false, false))
	require.Equal(t, int64(180_000), chargeAt(t, l, q(100), false, true), "within custody, after another pool")
	require.Equal(t, int64(400_000), chargeAt(t, l, q(150), false, true), "an After cell above its cell: the cell")
	require.Equal(t, int64(520_000), chargeAt(t, l, q(300), false, true), "a first borrow, after another pool")
	require.Equal(t, int64(600_000), chargeAt(t, l, q(300), false, false))
	require.Equal(t, int64(100_000), chargeAt(t, l, q(300), true, true), "the pool's own second fill")
	require.Equal(t, int64(100_000), chargeAt(t, l, q(300), true, false))

	gap := l
	gap.OwedOut = owedSome // a borrow while owing: no cell of the ladder allows it
	require.Equal(t, int64(250_000), chargeAt(t, gap, q(300), false, false), "the least cell")
	require.Equal(t, int64(180_000), chargeAt(t, gap, q(300), false, true), "the least cell after another pool")

	none := l
	none.After = nil
	for _, v := range []uint64{100, 150, 300} {
		require.Equal(t, chargeAt(t, none, q(v), false, false), chargeAt(t, none, q(v), false, true))
	}

	bad := l
	bad.After = []Cell{{0, 0}}
	bad.Points = []Point{{q(1), q(1)}}
	_, err := newCurve(bad)
	require.ErrorIs(t, err, ErrInvalidState)
}

// A ladder's cells after another pool survive its state's encoding, and a ladder without them encodes as before, with
// no key of theirs.
func TestTheCellsAfterAnotherPoolSurviveTheStatesEncoding(t *testing.T) {
	l := Ladder{In: 0, Out: 1, Points: pts(100, 200), Cells: []Cell{{0, 250_000}, {pathOut, 400_000}},
		Second: 100_000, After: []Cell{{0, 180_000}, {pathOut, 330_000}}}
	raw, err := json.Marshal(Extra{Ladders: []Ladder{l}})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"sc":100000,"ac":[{"p":0,"g":180000},{"p":1,"g":330000}]`)
	var back Extra
	require.NoError(t, json.Unmarshal(raw, &back))
	require.Equal(t, l.After, back.Ladders[0].After)
	l.After = nil
	raw, err = json.Marshal(Extra{Ladders: []Ladder{l}})
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"ac"`)
}

// Every cell after another pool is a cell of the table's own key and path, at or under it, over its key's second
// fill, a base rounded down to 1,000; and afterOf gives every path of a key, in the cells' order.
func TestTheTablesCellsAfterAnotherPoolAreUnderItsCells(t *testing.T) {
	require.NotEmpty(t, gasCellsAfter)
	for _, a := range gasCellsAfter {
		at := slices.IndexFunc(gasCells, func(c gasCell) bool {
			return strings.EqualFold(c.family, a.family) && c.chain == a.chain && c.kind == a.kind &&
				c.sale == a.sale && c.path == a.path
		})
		require.GreaterOrEqual(t, at, 0, "no cell for %+v", a)
		require.Positive(t, a.gas)
		require.Zero(t, a.gas%1_000, "a base is rounded down to 1,000")
		require.LessOrEqual(t, a.gas, gasCells[at].gas, "%+v", a)
		second, ok := secondOf(a.family, a.chain, a.kind, a.sale)
		require.True(t, ok)
		require.Less(t, second, a.gas, "%+v", a)
	}
	keys := 0
	for _, c := range gasCells {
		if !slices.ContainsFunc(gasFamilies, func(f HookFamily) bool { return strings.EqualFold(f.CodeHash, c.family) }) {
			continue // a test build's
		}
		cells, after := cellsOf(c.family, c.chain, c.kind, c.sale), afterOf(c.family, c.chain, c.kind, c.sale)
		require.Len(t, after, len(cells))
		for k := range cells {
			require.Equal(t, cells[k].Path, after[k].Path)
			require.LessOrEqual(t, after[k].Gas, cells[k].Gas)
		}
		keys++
	}
	require.Positive(t, keys)
	// a build with no cell after another pool: its cells
	require.Equal(t, cellsOf(testHookFamily, valueobject.ChainIDBase, 0, true),
		afterOf(testHookFamily, valueobject.ChainIDBase, 0, true))
}

// chargedAfterAtMost checks that a cell's fill after another pool is charged at most its net gas, as chargedAtMost's reads allow its path: with the
// route's limit saying another pool filled, the charge is at or under the fill's net, and under the charge alone.
func chargedAfterAtMost(t *testing.T, f cellFill) {
	t.Helper()
	q, zero := uint256.NewInt(1_000), uint256.NewInt(0)
	l := Ladder{Cells: cellsOf(f.family, f.chain, f.kind, f.sale), After: afterOf(f.family, f.chain, f.kind, f.sale),
		Held: zero, Freed: zero, OwedOut: owedNone, OwedIn: owedNone}
	require.True(t, slices.ContainsFunc(l.Cells, func(c Cell) bool { return c.Path == f.path }), f.name)
	if f.path&pathOut == 0 {
		l.Held = q
	}
	if f.path&(pathFirst|pathOwing) == 0 {
		l.Freed = q
	}
	if f.path&pathOwing != 0 {
		l.OwedOut = owedSome
	}
	if f.path&pathRepay != 0 {
		l.OwedIn, l.Repay = owedSome, true
	}
	g := chargeAt(t, l, q, false, true)
	require.LessOrEqual(t, g, f.net, "%s: charged %d", f.name, g)
	require.Less(t, g, chargeAt(t, l, q, false, false), f.name)
}

// cellFill is the least net gas measured of the fills of one cell, a key's path, in one state of their transaction.
type cellFill struct {
	measuredFill
	path uint8
}

// A charge never over the net gas measured: a pool's first fill after another pool of the source filled earlier in the same transaction, on another
// venue kind, the same pair and side: with the route's limit saying so, each cell's least net measured of such fills
// is charged at or under it. With no limit it is charged its cell alone, which is over some of these nets: the
// reason the limit is read.
func TestTheTableChargesAFillAfterAnotherPoolAtMostItsGas(t *testing.T) {
	over := 0
	fills := []cellFill{
		// generated: the least net of each cell's fills after another pool
		{measuredFill{"Base Morpho Blue sale, custody", measuredBuild, valueobject.ChainIDBase, 0, true, 150_346}, 0},
		{measuredFill{"Base Morpho Blue sale, a venue pays", measuredBuild, valueobject.ChainIDBase, 0, true, 270_025}, pathOut},
		{measuredFill{"Base Morpho Blue sale, first borrow", measuredBuild, valueobject.ChainIDBase, 0, true, 444_472}, pathOut | pathFirst},
		{measuredFill{"Base Morpho Blue sale, owing borrow", measuredBuild, valueobject.ChainIDBase, 0, true, 346_658}, pathOut | pathOwing},
		{measuredFill{"Base Morpho Blue buy, custody", measuredBuild, valueobject.ChainIDBase, 0, false, 147_868}, 0},
		{measuredFill{"Base Morpho Blue buy, repay", measuredBuild, valueobject.ChainIDBase, 0, false, 266_708}, pathRepay},
		{measuredFill{"Base Morpho Blue buy, repay, a venue pays", measuredBuild, valueobject.ChainIDBase, 0, false, 361_969}, pathOut | pathRepay},
		{measuredFill{"Base Aave V3 sale, custody", measuredBuild, valueobject.ChainIDBase, 1, true, 150_395}, 0},
		{measuredFill{"Base Aave V3 sale, a venue pays", measuredBuild, valueobject.ChainIDBase, 1, true, 456_089}, pathOut},
		{measuredFill{"Base Aave V3 sale, first borrow", measuredBuild, valueobject.ChainIDBase, 1, true, 929_365}, pathOut | pathFirst},
		{measuredFill{"Base Aave V3 sale, owing borrow", measuredBuild, valueobject.ChainIDBase, 1, true, 725_930}, pathOut | pathOwing},
		{measuredFill{"Base Aave V3 buy, custody", measuredBuild, valueobject.ChainIDBase, 1, false, 138_278}, 0},
		{measuredFill{"Base Aave V3 buy, repay", measuredBuild, valueobject.ChainIDBase, 1, false, 462_018}, pathRepay},
		{measuredFill{"Base Aave V3 buy, repay, a venue pays", measuredBuild, valueobject.ChainIDBase, 1, false, 756_768}, pathOut | pathRepay},
		{measuredFill{"Ethereum Morpho Blue sale, custody", measuredBuild, valueobject.ChainIDEthereum, 0, true, 146_726}, 0},
		{measuredFill{"Ethereum Morpho Blue sale, a venue pays", measuredBuild, valueobject.ChainIDEthereum, 0, true, 266_850}, pathOut},
		{measuredFill{"Ethereum Morpho Blue sale, first borrow", measuredBuild, valueobject.ChainIDEthereum, 0, true, 445_143}, pathOut | pathFirst},
		{measuredFill{"Ethereum Morpho Blue sale, owing borrow", measuredBuild, valueobject.ChainIDEthereum, 0, true, 344_283}, pathOut | pathOwing},
		{measuredFill{"Ethereum Morpho Blue buy, custody", measuredBuild, valueobject.ChainIDEthereum, 0, false, 144_369}, 0},
		{measuredFill{"Ethereum Morpho Blue buy, repay", measuredBuild, valueobject.ChainIDEthereum, 0, false, 263_960}, pathRepay},
		{measuredFill{"Ethereum Morpho Blue buy, repay, a venue pays", measuredBuild, valueobject.ChainIDEthereum, 0, false, 355_601}, pathOut | pathRepay},
		{measuredFill{"Ethereum Aave V3 sale, custody", measuredBuild, valueobject.ChainIDEthereum, 1, true, 146_775}, 0},
		{measuredFill{"Ethereum Aave V3 sale, a venue pays", measuredBuild, valueobject.ChainIDEthereum, 1, true, 433_545}, pathOut},
		{measuredFill{"Ethereum Aave V3 sale, first borrow", measuredBuild, valueobject.ChainIDEthereum, 1, true, 894_808}, pathOut | pathFirst},
		{measuredFill{"Ethereum Aave V3 sale, owing borrow", measuredBuild, valueobject.ChainIDEthereum, 1, true, 686_550}, pathOut | pathOwing},
		{measuredFill{"Ethereum Aave V3 buy, custody", measuredBuild, valueobject.ChainIDEthereum, 1, false, 134_779}, 0},
		{measuredFill{"Ethereum Aave V3 buy, repay", measuredBuild, valueobject.ChainIDEthereum, 1, false, 447_487}, pathRepay},
		{measuredFill{"Ethereum Aave V3 buy, repay, a venue pays", measuredBuild, valueobject.ChainIDEthereum, 1, false, 725_346}, pathOut | pathRepay},
		{measuredFill{"BSC Aave V3 sale, custody", bscBuild, valueobject.ChainIDBSC, 1, true, 139_680}, 0},
		{measuredFill{"BSC Aave V3 sale, a venue pays", bscBuild, valueobject.ChainIDBSC, 1, true, 418_573}, pathOut},
		{measuredFill{"BSC Aave V3 sale, first borrow", bscBuild, valueobject.ChainIDBSC, 1, true, 832_895}, pathOut | pathFirst},
		{measuredFill{"BSC Aave V3 sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 1, true, 631_111}, pathOut | pathOwing},
		{measuredFill{"BSC Aave V3 buy, custody", bscBuild, valueobject.ChainIDBSC, 1, false, 128_323}, 0},
		{measuredFill{"BSC Aave V3 buy, repay", bscBuild, valueobject.ChainIDBSC, 1, false, 431_977}, pathRepay},
		{measuredFill{"BSC Aave V3 buy, repay, a venue pays", bscBuild, valueobject.ChainIDBSC, 1, false, 667_197}, pathOut | pathRepay},
		{measuredFill{"BSC Venus Core sale, custody", bscBuild, valueobject.ChainIDBSC, 2, true, 256_932}, 0},
		{measuredFill{"BSC Venus Core sale, a venue pays", bscBuild, valueobject.ChainIDBSC, 2, true, 767_860}, pathOut},
		{measuredFill{"BSC Venus Core sale, first borrow", bscBuild, valueobject.ChainIDBSC, 2, true, 1_861_106}, pathOut | pathFirst},
		{measuredFill{"BSC Venus Core sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 2, true, 1_589_075}, pathOut | pathOwing},
		{measuredFill{"BSC Venus Core buy, custody", bscBuild, valueobject.ChainIDBSC, 2, false, 245_575}, 0},
		{measuredFill{"BSC Venus Core buy, repay", bscBuild, valueobject.ChainIDBSC, 2, false, 759_642}, pathRepay},
		{measuredFill{"BSC Venus Core buy, repay, a venue pays", bscBuild, valueobject.ChainIDBSC, 2, false, 1_585_281}, pathOut | pathRepay},
		{measuredFill{"BSC Lista Moolah sale, custody", bscBuild, valueobject.ChainIDBSC, 3, true, 154_214}, 0},
		{measuredFill{"BSC Lista Moolah sale, first borrow", bscBuild, valueobject.ChainIDBSC, 3, true, 708_540}, pathOut | pathFirst},
		{measuredFill{"BSC Lista Moolah sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 3, true, 599_222}, pathOut | pathOwing},
		{measuredFill{"BSC Lista Moolah buy, custody", bscBuild, valueobject.ChainIDBSC, 3, false, 142_857}, 0},
		{measuredFill{"BSC Lista Moolah buy, repay", bscBuild, valueobject.ChainIDBSC, 3, false, 394_765}, pathRepay},
		{measuredFill{"BSC Lista Moolah buy, repay, a venue pays", bscBuild, valueobject.ChainIDBSC, 3, false, 371_499}, pathOut | pathRepay},
		// end of the generated fills
	}
	require.NotEmpty(t, fills)
	for _, f := range fills {
		chargedAfterAtMost(t, f)
		for _, c := range cellsOf(f.family, f.chain, f.kind, f.sale) {
			if c.Path == f.path && c.Gas > f.net {
				over++
			}
		}
	}
	require.Positive(t, over, "no cell alone is over a fill after another pool")
}

// The tracker gives a ladder its cells after another pool from the table: the paths of its cells, each at the least
// of the pair's live kinds, a kind's path with no cell after another pool at its cell; and none where no live kind
// has one, so its state is what it was.
func TestTrackerGivesALadderItsCellsAfterAnotherPool(t *testing.T) {
	tracked := func(venues []venueView) entity.Pool {
		w, cfg := testWorld(), testConfig()
		w.pools[common.HexToAddress(poolX)].venues = venues
		n, pools := listed(t, w, cfg)
		p, err := track(t, n, cfg, pools[0])
		require.NoError(t, err)
		require.Len(t, extraOf(t, p).Ladders, 2)
		return p
	}
	ladders := func(venues []venueView) []Ladder { return extraOf(t, tracked(venues)).Ladders }
	one, both := []venueView{venue(0, 0, 1, 0)}, []venueView{venue(0, 0, 1, 0), venue(1, 0, 2, 0)}
	for _, l := range ladders(both) {
		require.Nil(t, l.After, "no cell after another pool in the table")
	}
	kept := gasCellsAfter
	t.Cleanup(func() { gasCellsAfter = kept })
	gasCellsAfter = append(slices.Clone(kept),
		gasCell{testHookFamily, valueobject.ChainIDBase, 0, true, 0, 150_000},
		gasCell{testHookFamily, valueobject.ChainIDBase, 0, true, pathOut, 330_000},
		gasCell{testHookFamily, valueobject.ChainIDBase, 0, true, pathOut | pathOwing, 999_000_000}, // over its cell: the cell
		gasCell{testHookFamily, valueobject.ChainIDBase, 1, true, 0, 140_000},
		gasCell{testHookFamily, valueobject.ChainIDBase, 1, true, pathOut | pathFirst, 610_000},
		gasCell{testHookFamily, valueobject.ChainIDBase, 0, true, pathRepay, 1_000}, // no cell of the path: unread
		gasCell{testHookFamily, valueobject.ChainIDBSC, 0, true, 0, 1_000},          // another chain's
		gasCell{testPairFamily, valueobject.ChainIDBase, 0, true, 0, 1_000})         // another build's
	want := func(kinds ...uint8) []Cell {
		cs := testCells(testHookFamily, true, kinds...)
		for k := range cs {
			switch {
			case cs[k].Path == 0 && slices.Contains(kinds, 1):
				cs[k].Gas = 140_000
			case cs[k].Path == 0:
				cs[k].Gas = 150_000
			case cs[k].Path == pathOut:
				cs[k].Gas = 330_000 // kind 0's, under kind 1's cell
			case cs[k].Path == pathOut|pathFirst && slices.Contains(kinds, 1):
				// kind 1's cell after another pool, 610,000, against kind 0's own cell, 600,000: the least
				cs[k].Gas = min(610_000, testGas(testHookFamily, 0, 600_000))
			}
		}
		return cs
	}
	for _, tc := range []struct {
		venues []venueView
		kinds  []uint8
	}{{one, []uint8{0}}, {both, []uint8{0, 1}}} {
		for _, l := range ladders(tc.venues) {
			require.Equal(t, testCells(testHookFamily, l.In == 0, tc.kinds...), l.Cells)
			if l.In != 0 {
				require.Nil(t, l.After, "the buy has no cell after another pool")
				continue
			}
			require.Equal(t, want(tc.kinds...), l.After, "kinds %v", tc.kinds)
		}
	}
	// and the simulator charges them: a pool tracked with them, quoted after a fill on another with the route's limit
	p := tracked(one)
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	other := limitPools(t)[0]
	so := other.sim(t)
	limit := routeLimit(s, so)
	other.apply(t, so, other.sale, limit)
	lowered := 0
	for _, l := range extraOf(t, p).Ladders {
		at := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[l.In].Address,
			Amount: l.Points[0][0].ToBig()}, TokenOut: p.Tokens[l.Out].Address}
		alone, err := s.CalcAmountOut(at)
		require.NoError(t, err)
		at.Limit = limit
		after, err := s.CalcAmountOut(at)
		require.NoError(t, err)
		sameButGas(t, alone, after, "tracked, then quoted after another pool")
		switch l.In {
		case 0:
			require.Equal(t, testGas(testHookFamily, 0, 400_000), alone.Gas, "the sale, past its custody: a venue pays")
			require.Equal(t, int64(330_000), after.Gas, "the path's cell after another pool")
			lowered++
		default:
			require.Equal(t, alone.Gas, after.Gas, "a path with no cell after another pool")
		}
	}
	require.Equal(t, 1, lowered)
}
