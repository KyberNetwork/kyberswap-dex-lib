package everlongflamm

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sort"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// testdata/base-fork-lever.json.gz is the calm fixture's two pools after three sales, so that the leverage route wins
// a small sale and the curve a large one. Near the top of the leverage route's band its answer rounds: over a few
// sizes the leverage route answers, refuses and answers again, and where it refuses the pool pays the curve's fill,
// about 3 % less (the far edge). The hooks run a build of no listed family, so the fixture's config names it. Beside
// the two refreshes the fixture holds fills through the adapter at sweepSizes of each direction's ladders, fills at
// every size across each sale's far edge and 200 either side, and a walk with two leverage rungs forced into that
// edge.

// forkSweep is one direction's fills through the adapter at the refresh block, at sweepSizes of its two refreshes'
// ladders in order: Out holds each size's output less the one before it (fills), the output 0 where the swap
// reverts. Where it fills, the pool takes the whole input.
type forkSweep struct {
	Pool     string  `json:"pool"`
	TokenIn  string  `json:"tokenIn"`
	TokenOut string  `json:"tokenOut"`
	Out      []int64 `json:"out"`
}

// forkEdge is a pool's sale, token 0 for token 1, across the far edge: fills at every size from From on, Out as
// forkSweep's, and two sizes the leverage route prices with one the curve fills between them.
type forkEdge struct {
	Pool  string   `json:"pool"`
	From  uint64   `json:"from"`
	Out   []int64  `json:"out"`
	Rungs []uint64 `json:"rungs"`
}

// fills is the outputs a forkSweep's or a forkEdge's Out holds.
func fills(steps []int64) []uint64 {
	out := make([]uint64, len(steps))
	var at int64
	for k, d := range steps {
		at += d
		out[k] = uint64(at)
	}
	return out
}

// sweepLog is how many sizes sweepSizes spaces about a constant ratio apart across a direction's ladders, sweepFlat
// how many it spaces evenly across each flat span, where the route may change.
const (
	sweepLog  = 2000
	sweepFlat = 300
)

// leverParams are the params() words of the fixture's hooks that the tolerance and rowOk read: the curve's A and
// spans (words 0-2) and the leverage row (27-34), each WAD.
var leverParams = map[int]string{0: "34000000000000000000", 1: "6000000000000000000", 2: "6000000000000000000",
	27: "17500000000000000", 28: "13000000000000000", 29: "5000000000000000", 30: "20000000000000000",
	31: "2200000000000000000", 32: "50000000000000000", 33: "0", 34: "1700000000000000000"}

// sweepSizes is the grid a direction's fills are checked at: sweepLog sizes about a constant ratio apart from the
// least first point of its ladders to their greatest top, every point's input and one unit either side, and
// sweepFlat sizes across each flat span. Integers only, so every machine draws the same grid.
func sweepSizes(ls []Ladder) []*big.Int {
	var lo, hi *big.Int
	var sizes []*big.Int
	for _, l := range ls {
		for k, p := range l.Points {
			x := p[0].ToBig()
			sizes = append(sizes, new(big.Int).Sub(x, one), x, new(big.Int).Add(x, one))
			if k == 0 || l.Flat>>k&1 == 0 {
				continue
			}
			a := l.Points[k-1][0].ToBig()
			step := maxBig(new(big.Int).Quo(new(big.Int).Sub(x, a), big.NewInt(sweepFlat)), one)
			for s := new(big.Int).Add(a, step); s.Cmp(x) < 0; s = new(big.Int).Add(s, step) {
				sizes = append(sizes, s)
			}
		}
		if len(l.Points) != 0 {
			first, top := l.Points[0][0].ToBig(), l.Points[len(l.Points)-1][0].ToBig()
			if lo == nil || first.Cmp(lo) < 0 {
				lo = first
			}
			if hi == nil || top.Cmp(hi) > 0 {
				hi = top
			}
		}
	}
	if lo != nil {
		sizes = append(sizes, logSizes(lo, hi, sweepLog)...)
	}
	slices.SortFunc(sizes, func(a, b *big.Int) int { return a.Cmp(b) })
	sizes = slices.CompactFunc(sizes, func(a, b *big.Int) bool { return a.Cmp(b) == 0 })
	return slices.DeleteFunc(sizes, func(a *big.Int) bool { return a.Sign() <= 0 })
}

// logSizes is at most n sizes from lo to hi, each the last plus its s-th part, one unit at least, with s the largest
// that reaches hi within n.
func logSizes(lo, hi *big.Int, n int) []*big.Int {
	walk := func(s *big.Int, keep bool) ([]*big.Int, bool) {
		var sizes []*big.Int
		x := new(big.Int).Set(lo)
		for steps := 0; x.Cmp(hi) < 0; steps++ {
			if steps == n-1 {
				return nil, false
			}
			if keep {
				sizes = append(sizes, new(big.Int).Set(x))
			}
			x.Add(x, maxBig(new(big.Int).Quo(x, s), one))
		}
		return append(sizes, new(big.Int).Set(hi)), true
	}
	a, b := big.NewInt(1), new(big.Int).Set(hi)
	for a.Cmp(b) < 0 {
		m := new(big.Int).Add(a, b)
		m.Add(m, one).Rsh(m, 1)
		if _, ok := walk(m, false); ok {
			a = m
		} else {
			b = m.Sub(m, one)
		}
	}
	sizes, _ := walk(a, true)
	return sizes
}

// forkTolerance is the most a quote may pass the fill by, in the output's base units, where a chord of either route
// meets both routes' rounding: the curve floors its input to a grid that the anchor's root and the book's depth
// scale, and the leverage route keeps three units of the output and its solve's tolerance. With ax = A (min(spanUp,
// spanDn) - 1) from params() words 0-2 and root the square root of token 0's price in token 1 at the feed, in whole
// units:
//   - a sale: 4 + 2U + (4e-18 + 2.5e-18 root) cv (1 + 1/ax), U a unit of token 0 in token 1's base units plus one, cv
//     the curve's value (levState word 0) in token 1's base units, or twice the book's value at the feed where it
//     reads none;
//   - a buy: 4 + 8e-18 (V + N + cvOut) (1 + 1/ax) (1 + 1/root), V token 0's held, supplied and posted legs, N the idle
//     input and cvOut the curve's value, both in token 0's base units;
//   - 4 where the request reads no price.
//
// The form is read from the pool's code, its constants fitted to runs; it bounds rounding, not a route's price.
func forkTolerance(sale bool, req []*big.Int, cvWad *big.Int, params []*big.Int, dIn, dOut uint8) *big.Int {
	usd0, usd1, d0, d1 := req[11], req[12], dIn, dOut
	if !sale {
		usd0, usd1, d0, d1 = req[12], req[11], dOut, dIn
	}
	p36 := new(big.Int)
	if usd1.Sign() != 0 {
		p36.Mul(usd0, pow10(int(d1))).Mul(p36, e36).Quo(p36, new(big.Int).Mul(usd1, pow10(int(d0))))
	}
	legs := func(k ...int) *big.Int {
		s := new(big.Int)
		for _, i := range k {
			s.Add(s, req[15+i])
		}
		return s
	}
	v := legs(4, 5, 6)
	unread := req[10].Sign() == 0 || usd0.Sign() == 0 || usd1.Sign() == 0 || p36.Sign() == 0 || !sale && v.Sign() == 0
	if unread && (sale || v.Sign() != 0 || p36.Sign() == 0) {
		return big.NewInt(4)
	}
	f := func(x *big.Int) float64 {
		r, _ := new(big.Float).SetInt(x).Float64()
		return r
	}
	ax := float64(f(params[0])/1e18) * (f(minBig(params[1], params[2]))/1e18 - 1)
	book := 1 + 1/ax
	root := math.Sqrt(float64(f(usd0)*math.Pow10(18-int(d0))) / f(usd1))
	cv := new(big.Int).Quo(cvWad, pow10(18-int(d1))) // levState's words are WAD-scaled
	if !sale {
		n := new(big.Int)
		if !unread {
			n.Sub(legs(0, 1), req[15+3])
			n.Mul(maxBig(n, new(big.Int)), e36).Quo(n, p36)
		}
		cvOut := cv.Mul(cv, e36).Quo(cv, p36)
		units := float64(float64(8e-18*f(n.Add(n.Add(n, v), cvOut)))*book) * (1 + 1/root)
		return big.NewInt(4 + int64(units))
	}
	if cvWad.Sign() == 0 {
		cv.Mul(legs(0, 1, 2), p36).Quo(cv, e36)
		cv.Add(cv, maxBig(new(big.Int).Sub(legs(4, 5), req[15+7]), new(big.Int))).Lsh(cv, 1)
	}
	u := new(big.Int).Quo(p36, e36)
	units := float64((4e-18+float64(2.5e-18*root))*f(cv)) * book
	return u.Add(u, one).Lsh(u, 1).Add(u, big.NewInt(4+int64(units)))
}

// leverReads are the words the tolerance reads, as the tracker reads them: the hook's params(), and the request the
// pool hands the hook for a swap of `amount` from the taker with the hook's levState on it.
func leverReads(ctx context.Context, c *ethrpc.Client, taker, pl, hook, in, out common.Address, amount,
	block *big.Int) (params, req, lev []*big.Int, err error) {
	call := func(from, to common.Address, data []byte, words int) ([]*big.Int, error) {
		ret, err := c.GetETHClient().CallContract(ctx, ethereum.CallMsg{From: from, To: &to, Data: data}, block)
		if err != nil {
			return nil, err
		}
		if len(ret) != 32*words {
			return nil, ErrRead
		}
		return lo.Times(words, func(k int) *big.Int { return word(ret, k) }), nil
	}
	data, err := hookABI.Pack("params")
	if err != nil {
		return nil, nil, nil, err
	}
	if params, err = call(common.Address{}, hook, data, paramsWords); err != nil {
		return nil, nil, nil, err
	}
	if data, err = poolABI.Pack("pairRequest", in, out, amount); err != nil {
		return nil, nil, nil, err
	}
	if req, err = call(taker, pl, data, requestWords); err != nil {
		return nil, nil, nil, err
	}
	raw := slices.Concat(lo.Map(req, func(w *big.Int, _ int) []byte { return common.BigToHash(w).Bytes() })...)
	lev, err = call(common.Address{}, hook, slices.Concat(levStateSelector, raw), levStateWords)
	return params, req, lev, err
}

// forcedAsk is a walk's ask with its seeds from `from` up left out and rungs in their place, and none above them.
func forcedAsk(a sampleAsk, from uint64, rungs []uint64) sampleAsk {
	edge := new(big.Int).SetUint64(from)
	a.Seeds = slices.Concat(lo.Filter(a.Seeds, func(s *big.Int, _ int) bool { return s.Cmp(edge) < 0 }),
		lo.Map(rungs, func(r uint64, _ int) *big.Int { return new(big.Int).SetUint64(r) }))
	return a
}

// leverPool is what the lever tests read of a pool at its refresh block, from the fixture's calls: the replayed
// tracker, the pool's state and directions, and each direction's tolerance. On the way it checks what the tolerance
// stands on: params() words where the tracker reads the row, a row that passes rowOk, and a request the pool hands
// the hook naming the taker as its caller.
type leverPool struct {
	tr   *PoolTracker
	st   *poolState
	dirs []direction
	tol  map[[2]int]*big.Int
}

func leverPoolOf(t *testing.T, fx *forkFixture, p entity.Pool) leverPool {
	t.Helper()
	ctx := context.Background()
	cfg := fx.Config
	n := replayNode(t, fx, p.BlockNumber)
	tr := NewPoolTracker(&cfg, n.client())
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	st, err := tr.read(ctx, p.Address, se.Factory, len(p.Tokens), nil)
	require.NoError(t, err)
	require.NoError(t, tr.readVenues(ctx, p.Address, st, nil))
	dirs, err := tr.directions(ctx, se.Factory, len(p.Tokens), st, nil)
	require.NoError(t, err)
	lp := leverPool{tr: tr, st: st, dirs: dirs, tol: map[[2]int]*big.Int{}}
	live := new(big.Int).SetUint64(st.tokens[1].Config.SwapPriceBandWad)
	for _, l := range extraOf(t, p).Ladders {
		d, ok := lo.Find(dirs, func(d direction) bool { return d.in == int(l.In) && d.out == int(l.Out) })
		require.True(t, ok)
		params, req, lev, err := leverReads(ctx, n.client(), cfg.taker(), common.HexToAddress(p.Address), d.hook,
			st.tokens[d.in].Config.Token, st.tokens[d.out].Config.Token, l.Points[0][0].ToBig(), st.block)
		require.NoError(t, err)
		for k, want := range leverParams {
			require.Equal(t, want, params[k].String(), "params() word %d", k)
		}
		row := []*big.Int{d.row.spread, d.row.spreadDn, d.row.upCap, d.row.dnCap, d.row.ceiling, d.row.band,
			d.row.feedGate, d.row.upFloor}
		require.Equal(t, lo.Map(params[27:35], func(w *big.Int, _ int) string { return w.String() }),
			lo.Map(row, func(w *big.Int, _ int) string { return w.String() }), "the row the tracker read")
		require.True(t, rowOk(d.row, live) && d.trusted, "the deployed row passes rowOk")
		require.Equal(t, cfg.taker(), common.BigToAddress(req[0]), "the caller the pool hands the hook")
		lp.tol[[2]int{d.in, d.out}] = forkTolerance(d.in == 0, req, lev[0], params, st.tokens[d.in].Config.Decimals,
			st.tokens[d.out].Config.Decimals)
	}
	return lp
}

// checkSweeps checks each direction's fills at sweepSizes of its two ladders: every quote of either refresh at most
// the fill plus the tolerance with the whole input used, or the pool reverts; and so are each ladder's quote with no
// haircut and the rule's own chord (chordOf), which the bound must hold without.
func checkSweeps(t *testing.T, fx *forkFixture, sims []map[string]*PoolSimulator) {
	for _, sw := range fx.Sweeps {
		k := slices.IndexFunc(fx.Pools, func(p entity.Pool) bool { return p.Address == sw.Pool })
		require.GreaterOrEqual(t, k, 0, sw.Pool)
		p := fx.Pools[k]
		index := func(a string) int {
			return slices.IndexFunc(p.Tokens, func(tk *entity.PoolToken) bool { return tk.Address == a })
		}
		in, out := index(sw.TokenIn), index(sw.TokenOut)
		tol := leverPoolOf(t, fx, p).tol[[2]int{in, out}]
		require.NotNil(t, tol)
		ladders := lo.FilterMap([]entity.Pool{p, fx.Warm[k]}, func(q entity.Pool, _ int) (Ladder, bool) {
			return lo.Find(extraOf(t, q).Ladders, func(l Ladder) bool { return int(l.In) == in && int(l.Out) == out })
		})
		sizes, paid := sweepSizes(ladders), fills(sw.Out)
		require.Len(t, paid, len(sizes), "the sweep's sizes")
		var quoted, unquoted, reverts int
		worst, bare, rule := new(big.Int).Neg(tol), new(big.Int).Neg(tol), new(big.Int).Neg(tol)
		for r := range sims {
			c, err := newCurve(quotingLadder(ladders[r]))
			require.NoError(t, err)
			for i, a := range sizes {
				res, err := sims[r][sw.Pool].CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: sw.TokenIn, Amount: a}, TokenOut: sw.TokenOut})
				switch {
				case paid[i] == 0:
					reverts++
					continue
				case err != nil:
					unquoted++
					continue
				}
				quoted++
				require.Zero(t, res.RemainingTokenAmountIn.Amount.Sign(), "refresh %d at %s", r+1, a)
				over := new(big.Int).Sub(res.TokenAmountOut.Amount, new(big.Int).SetUint64(paid[i]))
				require.LessOrEqual(t, over.Cmp(tol), 0, "%s, refresh %d at %s: %s over the fill, tolerance %s",
					fx.Kinds[sw.Pool], r+1, a, over, tol)
				worst = maxBig(worst, over)
				var used, chord uint256.Int
				_, err = c.quote(uint256.MustFromBig(a), 0, &used, &chord)
				require.NoError(t, err)
				over.Sub(chord.ToBig(), new(big.Int).SetUint64(paid[i]))
				require.LessOrEqual(t, over.Cmp(tol), 0, "%s, refresh %d at %s with no haircut: %s over the fill, "+
					"tolerance %s", fx.Kinds[sw.Pool], r+1, a, over, tol)
				bare = maxBig(bare, over)
				q, ok := chordOf(ladders[r], a)
				require.True(t, ok)
				over.Sub(q, new(big.Int).SetUint64(paid[i]))
				require.LessOrEqual(t, over.Cmp(tol), 0, "%s, refresh %d at %s, the rule's chord: %s over the fill, "+
					"tolerance %s", fx.Kinds[sw.Pool], r+1, a, over, tol)
				rule = maxBig(rule, over)
			}
		}
		t.Logf("%s, %d -> %d: %d sizes, %d quotes at most %s units over the fill, %s with no haircut and %s by the "+
			"rule's chord (tolerance %s), %d unquoted, %d where the pool reverts", fx.Kinds[sw.Pool], in, out,
			len(sizes), quoted, worst, bare, rule, tol, unquoted, reverts)
	}
}

// settleInOf is what build settles a direction's walk with, for the checks and the controls that settle it again.
func settleInOf(d direction, st *poolState, w walk) settleIn {
	return settleIn{probes: w.probes, words: w.words, rule: d.rule, tiers: d.row.tiers(), rowBand: d.row.band,
		sale: d.in == 0, decIn: st.tokens[d.in].Config.Decimals, decOut: st.tokens[d.out].Config.Decimals,
		trusted: d.trusted, curveFee: w.curveFee}
}

// lessAbove is the walk less its probe above: the one asked at used + (used >> aboveShift) + 1 of a priced,
// leverage-labelled probe. false: the walk holds no such probe.
func lessAbove(w walk, tiers []*big.Int) (walk, bool) {
	rest := lo.Reject(w.probes, func(q sampled, _ int) bool {
		return slices.ContainsFunc(w.probes, func(top sampled) bool {
			ask := new(big.Int).Rsh(top.used, aboveShift)
			return top.status == statusPriced && isTierOf(tiers, top.fee) &&
				q.ask.Cmp(ask.Add(ask, top.used).Add(ask, one)) == 0
		})
	})
	n := len(w.probes) - len(rest)
	w.probes = rest
	return w, n == 1
}

// tierPointsStand checks each leverage-labelled point the walk settles to: room inside the band past its margin,
// and a priced probe at least used >> aboveShift above it among the points taken once from the probes the dust and
// room drops leave, not one past where those points end. It returns how many there are.
func tierPointsStand(t *testing.T, name string, d direction, st *poolState, w walk) int {
	t.Helper()
	in := settleInOf(d, st, w)
	s, r := settle(in), newRoom(in)
	isTier := func(f *big.Int) bool { return isTierOf(in.tiers, f) }
	left := lo.Map(w.probes, func(q sampled, _ int) sampled {
		q.fee = orZero(q.fee)
		return q
	})
	if slices.ContainsFunc(left, func(q sampled) bool { return live(q) && q.out.Cmp(dust) >= 0 }) {
		left = slices.DeleteFunc(left, func(q sampled) bool { return live(q) && q.out.Cmp(dust) < 0 })
	}
	if r != nil {
		left = slices.DeleteFunc(left, func(q sampled) bool {
			return live(q) && isTier(q.fee) && !r.passes(q.used, q.out)
		})
	}
	_, _, top := points(left, isTier)
	var n int
	for i, x := range s.x {
		if !isTier(s.f[i]) {
			continue
		}
		n++
		probe, ok := lo.Find(w.probes, func(q sampled) bool { return live(q) && q.used.Cmp(x) == 0 })
		require.True(t, ok && r != nil && r.passes(probe.used, probe.out), "%s: a tier point at %s inside its margin",
			name, x)
		above := new(big.Int).Rsh(x, aboveShift)
		above.Add(above, x)
		require.True(t, top.Cmp(above) >= 0, "%s: a tier point at %s with no priced point of its ladder above it",
			name, x)
	}
	return n
}

// chordOf is the ladder's quote at a by the rule alone, with no haircut: a point's output at the point, the point
// below on a flat span, the chord's floor on any other, the top past a capped top; false under the first point and
// past a top not capped. It leaves out the simulator's guard, which quotes the point below on a span whose
// neighbours are not concave in the input: the bound must hold without it.
func chordOf(l Ladder, a *big.Int) (*big.Int, bool) {
	m := len(l.Points)
	if m == 0 {
		return nil, false
	}
	if a.Cmp(l.Points[m-1][0].ToBig()) > 0 {
		return l.Points[m-1][1].ToBig(), l.Capped
	}
	k := sort.Search(m, func(i int) bool { return l.Points[i][0].ToBig().Cmp(a) >= 0 })
	x1, y1 := l.Points[k][0].ToBig(), l.Points[k][1].ToBig()
	if x1.Cmp(a) == 0 {
		return y1, true
	}
	if k == 0 {
		return nil, false
	}
	x0, y0 := l.Points[k-1][0].ToBig(), l.Points[k-1][1].ToBig()
	if l.Flat>>k&1 == 1 {
		return y0, true
	}
	q := new(big.Int).Sub(a, x0)
	q.Mul(q, new(big.Int).Sub(y1, y0)).Quo(q, new(big.Int).Sub(x1, x0))
	return q.Add(q, y0), true
}

// worstOver is the most the ladder quotes past the fill at the sizes filled, and where, with no haircut: the
// simulator's quote, or with raw the rule's own chord (chordOf). The bound must hold without either. Sizes the
// ladder does not quote, or the pool refuses, are left out.
func worstOver(t *testing.T, l Ladder, filled map[uint64]uint64, raw bool) (*big.Int, uint64) {
	t.Helper()
	c, err := newCurve(l)
	require.NoError(t, err)
	worst, at := big.NewInt(math.MinInt64), uint64(0)
	for a, fill := range filled {
		var used, out uint256.Int
		q, ok := chordOf(l, new(big.Int).SetUint64(a))
		if !raw {
			_, err := c.quote(uint256.NewInt(a), 0, &used, &out)
			q, ok = out.ToBig(), err == nil
		}
		if !ok || fill == 0 {
			continue
		}
		if over := new(big.Int).Sub(q, new(big.Int).SetUint64(fill)); over.Cmp(worst) > 0 ||
			over.Cmp(worst) == 0 && a < at {
			worst, at = over, a
		}
	}
	return worst, at
}

// At the far edge the leverage route answers by rounding, so a leverage-labelled point there may stand on an answer
// the pool refuses a unit on, while the curve fills about 3 % less around it. A leverage-labelled point stands only
// with room inside the band past its margin and a priced probe above it among the ladder's own points: on both
// refreshes' walks of each direction and on a sale's walk with two leverage rungs forced into the edge, none
// published lacks either, and no quote passes the fill by more than the tolerance. With the room test and the probe
// above off, the forced walk's chord passes the fill by about 3 %: the state is armed.
func TestForkLeverEdgeIsNeverQuotedOverThePool(t *testing.T) {
	fx := loadFixture(t, "testdata/base-fork-lever.json.gz")
	ctx := context.Background()
	require.Len(t, fx.Edges, len(fx.Pools))
	for k, p := range fx.Pools {
		e, ok := lo.Find(fx.Edges, func(e forkEdge) bool { return e.Pool == p.Address })
		require.True(t, ok, p.Address)
		lp := leverPoolOf(t, fx, p)
		walksOf := func(asks []sampleAsk) []walk {
			capGas := lp.tr.defaultCap()
			a, err := lp.tr.askSampler(ctx, common.HexToAddress(p.Address), asks, capGas, lp.st.block, nil)
			require.NoError(t, err)
			walks, err := lp.tr.walksOf(p.Address, capGas, capGas, a)
			require.NoError(t, err)
			return walks
		}
		last := map[[2]uint8]Ladder{}
		for _, l := range extraOf(t, p).Ladders {
			last[[2]uint8{l.In, l.Out}] = l
		}
		asks := lp.tr.asks(lp.st, lp.dirs, nil)
		first, steady := walksOf(asks), walksOf(lp.tr.asks(lp.st, lp.dirs, last))
		for di, d := range lp.dirs {
			tol := lp.tol[[2]int{d.in, d.out}]

			// the fills by size: the direction's sweep, and on the sale the edge
			sw, ok := lo.Find(fx.Sweeps, func(s forkSweep) bool {
				return s.Pool == p.Address && s.TokenIn == p.Tokens[d.in].Address
			})
			require.True(t, ok)
			ladders := lo.FilterMap([]entity.Pool{p, fx.Warm[k]}, func(q entity.Pool, _ int) (Ladder, bool) {
				return lo.Find(extraOf(t, q).Ladders, func(l Ladder) bool { return int(l.In) == d.in && int(l.Out) == d.out })
			})
			require.Len(t, ladders, 2)
			sizes, paid := sweepSizes(ladders), fills(sw.Out)
			require.Len(t, paid, len(sizes))
			filled := map[uint64]uint64{}
			for i, a := range sizes {
				filled[a.Uint64()] = paid[i]
			}
			walks := []struct {
				name string
				w    walk
				want *Ladder
			}{{"first refresh", first[di], &ladders[0]}, {"second refresh", steady[di], &ladders[1]}}
			if d.in == 0 {
				for i, o := range fills(e.Out) {
					filled[e.From+uint64(i)] = o
				}
				walks = append(walks, struct {
					name string
					w    walk
					want *Ladder
				}{"rungs forced into the edge", walksOf([]sampleAsk{forcedAsk(asks[di], e.From, e.Rungs)})[0], nil})
			}
			for _, wk := range walks {
				name := fmt.Sprintf("%s, %d -> %d, %s", fx.Kinds[p.Address], d.in, d.out, wk.name)
				l, ok := build(d, lp.st, wk.w, wk.w.words)
				require.True(t, ok, name)
				if wk.want != nil {
					got := l
					if len(wk.want.Cells) == 0 { // recorded before the charge
						got = uncharged(l)
					}
					require.Equal(t, *wk.want, got, "%s: the ladder the refresh published", name)
				}
				tiers := tierPointsStand(t, name, d, lp.st, wk.w)
				over, at := worstOver(t, l, filled, false)
				require.LessOrEqual(t, over.Cmp(tol), 0, "%s at %d: %s over the fill, tolerance %s", name, at, over, tol)
				chord, cat := worstOver(t, l, filled, true)
				require.LessOrEqual(t, chord.Cmp(tol), 0, "%s at %d, the rule's chord: %s over the fill, tolerance %s",
					name, cat, chord, tol)
				t.Logf("%s: %d probes, %d points, %d of them tier-labelled, top %v, at most %s units over the fill, %s "+
					"by the rule's chord (tolerance %s, %d sizes)", name, len(wk.w.probes), len(l.Points), tiers,
					l.Points[len(l.Points)-1], over, chord, tol, len(filled))
			}
			if d.in != 0 {
				continue
			}

			// the control: the forced walk with its probe above left out, settled with the room test and the probe
			// above off, publishes a leverage-labelled top inside the edge whose chord passes the fill
			ctl, ok := lessAbove(walks[2].w, d.row.tiers())
			require.True(t, ok, "the walk's probe above")
			in := settleInOf(d, lp.st, ctl)
			in.roomTest, in.skipAbove = func(*room, sampled) bool { return true }, true
			s := settle(in)
			require.True(t, len(s.x) != 0 && isTierOf(in.tiers, s.f[len(s.f)-1]), "the control's top is leverage-labelled")
			l := ladderOf(t, s)
			over, at := worstOver(t, l, filled, false)
			require.Positive(t, over.Cmp(tol), "%s: the control does not pass the fill, the edge is not armed",
				fx.Kinds[p.Address])
			bps, _ := new(big.Float).Quo(new(big.Float).SetInt(over), new(big.Float).SetUint64(filled[at])).Float64()
			t.Logf("%s, control: top %v, at %d %s units over the fill, %.2f bp", fx.Kinds[p.Address],
				l.Points[len(l.Points)-1], at, over, bps*10_000)
		}
	}
}
