package efficientprop

//
// A property test of the branch's one invariant: the simulator never quotes an output above what the listing door pays
// for the same input in the same block, for a first fill and for every later fill of a route.
//
// The chain is chain_model_test.go (a port of the Solidity with math/big alone). The tracker's refresh is rebuilt
// here from the package's own grid functions (probeSizes, withSize, withEdges, rungEdges, minSizes, collectClassedRun,
// repeatOf), fed the model's answers, and the package's simulator is then asked for routes whose fills the model runs
// in sequence, on the pool's lane as FLAMMRoute books it.
//
// Run: CHAIN_LISTINGS=150000 CHAIN_WORKERS=8 go test ./pkg/liquidity-source/efficient-prop/ -run TestChainProperty -v

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const chNow0 = uint64(1_764_000_000_000)

type chListing struct {
	bd, qd     uint8
	base0      bool
	pair, word *big.Int
	ctl, s1    *big.Int
	bid, ask   *chSide
	pool       *chPool
	class      string
	extreme    bool
}

func (l *chListing) side(dir int) *chSide {
	if (dir == 0) == l.base0 {
		return l.bid
	}
	return l.ask
}

// reserves are the pool's custody by token index.
func (l *chListing) reserves() [2]*big.Int {
	if l.base0 {
		return [2]*big.Int{l.pool.held[0], l.pool.held[1]}
	}
	return [2]*big.Int{l.pool.held[1], l.pool.held[0]}
}

// chain is a copy of the listing a route's fills move.
func (l *chListing) chain() *chListing {
	c := *l
	c.pool = l.pool.clone()
	b, a := *l.bid, *l.ask
	b.pool, a.pool = c.pool, c.pool
	c.bid, c.ask = &b, &a
	return &c
}

func chOr(w *big.Int, lo uint, v *big.Int) { w.Or(w, new(big.Int).Lsh(v, lo)) }

func chRandBelow(rng *rand.Rand, n *big.Int) *big.Int {
	if n.Sign() <= 0 {
		return new(big.Int)
	}
	return new(big.Int).Rand(rng, n)
}

// chBetween is a random amount in [lo%, hi%] of v, in hundredths.
func chBetween(rng *rand.Rand, v *big.Int, lo, hi int64) *big.Int {
	p := lo + rng.Int63n(hi-lo+1)
	r := new(big.Int).Mul(v, big.NewInt(p))
	return r.Quo(r, big.NewInt(100))
}

func chEnc24(m *big.Int) uint64 {
	v, e := new(big.Int).Set(m), uint64(0)
	for v.Cmp(big.NewInt(262143)) > 0 {
		v.Quo(v, big.NewInt(10))
		e++
	}
	return v.Uint64() | e<<18
}

func chBoardExp(rng *rand.Rand, dec uint8) uint64 {
	base := uint64(0)
	if dec > 6 {
		base = uint64(dec) - 6
	}
	switch rng.Intn(6) {
	case 0:
		return min(16, base+uint64(rng.Intn(4)))
	case 1:
		return uint64(rng.Intn(int(min(16, uint64(dec))) + 1))
	}
	return base
}

// chDepth is a depth post the front's _checkDepth takes (PropAMMLandingCore :347-369): sizes rise strictly, offsets
// never fall, rung 0's offset is 0, the exponent at most 30, the depth under 2^40 board units. total is its depth in
// board units.
func chDepth(rng *rand.Rand, din uint8, exp uint64) (w0, w1 *big.Int, nonce uint64, total uint64, edges []uint64) {
	logB := float64(din) - float64(exp) + rng.Float64()*9 - 3
	logB = math.Max(0.4, math.Min(11.99, logB))
	b := math.Pow(10, logB)
	dExp := uint64(0)
	for b/math.Pow(10, float64(dExp)) > 65535 {
		dExp++
	}
	if rng.Intn(4) == 0 && b/math.Pow(10, float64(dExp+1)) >= 12 {
		dExp++
	}
	smax := uint64(b / math.Pow(10, float64(dExp)))
	smax = max(1, min(65535, smax))
	nr := 2 + rng.Intn(8)
	switch r := rng.Intn(20); {
	case r < 3:
		nr = 1
	case r < 9:
		nr = 10
	}
	var sizes, offs []uint64
	size, off := uint64(0), uint64(0)
	for j := 0; j < nr; j++ {
		inc := uint64(1)
		if rng.Intn(4) != 0 {
			inc = 1 + uint64(rng.Int63n(int64(max(1, 2*smax/uint64(nr)))))
		}
		size += inc
		if size > smax {
			break
		}
		if j > 0 {
			switch r := rng.Intn(10); {
			case r < 3:
			case r < 8:
				off += 1 + uint64(rng.Intn(300))
			case r < 9:
				off += 1 + uint64(rng.Intn(20000))
			default:
				off += 1 + uint64(rng.Intn(400000))
			}
			off = min(off, 1<<20-1)
		}
		sizes, offs = append(sizes, size), append(offs, off)
	}
	if len(sizes) == 0 {
		sizes, offs = []uint64{1}, []uint64{0}
	}
	nonce = 1 + uint64(rng.Int63n(1<<48-1))
	w0, w1 = big.NewInt(1), new(big.Int)
	chOr(w0, 12, chN(nonce))
	chOr(w0, 60, chN(2_000_000_000))
	chOr(w0, 112, chN(dExp))
	if len(sizes) > 3 {
		w0.SetBit(w0, 118, 1)
		w1.SetBit(w1, 252, 1)
	}
	scale := uint64(math.Pow(10, float64(dExp)) + 0.5)
	for j := range sizes {
		rg := chN(sizes[j] | offs[j]<<16)
		if j < 3 {
			chOr(w0, 120+36*uint(j), rg)
		} else {
			chOr(w1, 36*uint(j-3), rg)
		}
		edges = append(edges, sizes[j]*scale)
	}
	return w0, w1, nonce, sizes[len(sizes)-1] * scale, edges
}

// chGen is a random listing with no credit mode on a lane pair, and the blocks a refresh samples it in.
func chGen(rng *rand.Rand) (*chListing, []chBlock) {
	l := &chListing{}
	decs := [][2]uint8{{18, 6}, {18, 6}, {18, 6}, {18, 18}, {18, 18}, {8, 6}, {8, 6}, {6, 6}, {6, 6}, {18, 8},
		{6, 18}, {8, 18}, {8, 8}}
	d := decs[rng.Intn(len(decs))]
	l.bd, l.qd = d[0], d[1]
	l.base0 = rng.Intn(2) == 0
	l.extreme = rng.Intn(25) == 0 || chExtreme

	// the anchor, price36: quote base units for 1e36 base base units
	k := rng.Intn(10) - 3
	if l.extreme {
		k = 12 + rng.Intn(10)
		if rng.Intn(2) == 0 {
			k = -8 - rng.Intn(12)
		}
	}
	e := 36 + int(l.qd) - int(l.bd) - 6 + k
	e = min(e, 52)
	a := big.NewInt(100000 + rng.Int63n(9900000))
	if e >= 0 {
		a.Mul(a, chPow10(uint64(e)))
	} else {
		a.Quo(a, chPow10(uint64(-e)))
	}
	if a.Sign() == 0 {
		a.SetInt64(1)
	}

	eb, eq := chBoardExp(rng, l.bd), chBoardExp(rng, l.qd)
	l.pair = new(big.Int)
	if l.base0 {
		l.pair.SetBit(l.pair, 8, 1)
	}
	chOr(l.pair, 112, chN(eb))
	chOr(l.pair, 120, chN(eq))

	bw0, bw1, bn, btot, bedges := chDepth(rng, l.bd, eb)
	aw0, aw1, an, atot, aedges := chDepth(rng, l.qd, eq)
	// each side's depth in base units of its input, and on its quote leg
	bidIn := new(big.Int).Mul(chN(btot), chPow10(eb))
	askIn := new(big.Int).Mul(chN(atot), chPow10(eq))
	bidQuote := new(big.Int).Mul(bidIn, a)
	bidQuote.Quo(bidQuote, chE36)
	askBase := new(big.Int).Mul(askIn, chE36)
	askBase.Quo(askBase, a)

	// the print and the band
	P, conf := new(big.Int), uint64(0)
	l.s1 = new(big.Int)
	if rng.Intn(20) < 9 {
		P.Mul(a, big.NewInt(1_000_000+int64(rng.Intn(16001))-8000)).Quo(P, chPPM)
		fb, fa := 1+uint64(rng.Intn(2000)), 1+uint64(rng.Intn(2000))
		ceil := max(fb, fa) + uint64(rng.Intn(3000))
		chOr(l.s1, 0, chN(fb))
		chOr(l.s1, 16, chN(fa))
		chOr(l.s1, 32, chN(ceil))
		chOr(l.s1, 232, chN(uint64(rng.Intn(13))))
		if rng.Intn(2) == 0 {
			conf = uint64(rng.Intn(int(ceil*10) + 1))
		}
	}
	// the minimum fill, on the quote leg
	minLeg := chMinB(bidQuote, askIn)
	if rng.Intn(10) >= 3 && minLeg.Sign() > 0 {
		m := new(big.Int).Quo(minLeg, chPow10(uint64(1+rng.Intn(4))))
		chOr(l.s1, 160, chN(chEnc24(m)))
	}
	if rng.Intn(8) == 0 {
		chOr(l.s1, 184, chN(uint64(1+rng.Intn(10))))
	}

	// the word: its maker time and each side's refill ask
	ts := chNow0 - 1 - uint64(rng.Intn(8000))
	switch r := rng.Intn(20); {
	case r < 5:
		ts = chNow0 + uint64(rng.Intn(3000))
	case r < 8:
		ts = chNow0 + uint64(rng.Intn(40000))
	}
	l.word = new(big.Int)
	chOr(l.word, 72, chN(ts))
	l.ctl = big.NewInt(1)
	if rng.Intn(5) < 3 {
		l.ctl.SetBit(l.ctl, 66, 1)
	}

	// the pool
	l.pool = chGenPool(rng, l, askBase, bidQuote)

	side := func(bid bool, w0, w1 *big.Int, nonce, total uint64, edges []uint64, leg *big.Int) *chSide {
		s := &chSide{bid: bid, w0: w0, w1: w1, word: l.word, ctl: l.ctl, pair: l.pair, s1: l.s1, a: a, P: P, conf: conf,
			pool: l.pool}
		b := big.NewInt(1)
		tag := nonce
		if rng.Intn(7) == 0 {
			tag ^= 1
		}
		chOr(b, 4, chN(tag))
		filled := uint64(0)
		switch r := rng.Intn(10); {
		case r < 4:
		case r < 7:
			filled = uint64(rng.Int63n(int64(total)))
		case r < 9:
			filled = edges[rng.Intn(len(edges))]
		default:
			filled = total - 1
			if rng.Intn(2) == 0 {
				filled = min(1, total)
			}
		}
		chOr(b, 52, chN(filled))
		// the desk's bucket, on the quote leg
		m96 := new(big.Int).Sub(new(big.Int).Lsh(chOne, 96), chOne)
		s.bBurst, s.bRate = new(big.Int).Set(m96), new(big.Int)
		level := new(big.Int).Set(m96)
		if rng.Intn(5) < 2 && leg.Sign() > 0 {
			s.bBurst = chMinB(m96, new(big.Int).Add(chBetween(rng, leg, 2, 120), chOne))
			level = new(big.Int).Set(s.bBurst)
			if rng.Intn(2) == 0 {
				level = chRandBelow(rng, new(big.Int).Add(s.bBurst, chOne))
			}
			if rng.Intn(2) == 0 {
				s.bRate.Quo(s.bBurst, big.NewInt(5+rng.Int63n(600)))
			}
		}
		chOr(b, 92, level)
		seq := uint64(rng.Intn(2048))
		bt := ts - 1 - uint64(rng.Intn(100000))
		if rng.Intn(10) < 3 {
			bt = ts + uint64(rng.Intn(5000))
		}
		chOr(b, 188, chN(bt))
		chOr(b, 236, chN(seq))
		chOr(b, 247, chN(uint64(rng.Intn(512))))
		s.board = b
		// the word's ask to refill this side
		if rng.Intn(2) == 0 {
			ack := uint64(rng.Intn(2048))
			switch r := rng.Intn(10); {
			case r < 4:
				ack = seq
			case r < 7:
				ack = (seq + 1) & 0x7ff
			}
			if bid {
				l.word.SetBit(l.word, 232, 1)
				chOr(l.word, 234, chN(ack))
			} else {
				l.word.SetBit(l.word, 233, 1)
				chOr(l.word, 245, chN(ack))
			}
		}
		return s
	}
	l.bid = side(true, bw0, bw1, bn, btot, bedges, bidQuote)
	l.ask = side(false, aw0, aw1, an, atot, aedges, askIn)

	// the blocks a refresh samples: the pinned one, then up to two more
	c0 := uint64(0)
	switch r := rng.Intn(20); {
	case r < 4:
	case r < 12:
		c0 = uint64(rng.Intn(500))
	case r < 18:
		c0 = uint64(rng.Intn(20000))
	default:
		c0 = uint64(rng.Intn(990000))
	}
	blocks := []chBlock{{c: c0, nowMs: chNow0}}
	step := []uint64{12000, 12000, 1000, 250}[rng.Intn(4)]
	for i := 1; i < []int{1, 1, 2, 2, 3}[rng.Intn(5)]; i++ {
		c := c0
		switch r := rng.Intn(20); {
		case r < 7:
		case r < 12:
			c += 1 + uint64(rng.Intn(50))
		case r < 16:
			c += 50 + uint64(rng.Intn(5000))
		default:
			c -= min(c, 1+uint64(rng.Intn(5000)))
		}
		blocks = append(blocks, chBlock{c: c, nowMs: chNow0 + step*uint64(i)})
	}
	return l, blocks
}

// chGenPool is the listing's pool: one that bounds nothing, one set as the launch pool is (no spread rate, no
// notional rate, bucket units of 10^(decimals-6), with a custody, a bucket or a cap that may cut inside a side), or
// one with a spread rate and a notional rate.
func chGenPool(rng *rand.Rand, l *chListing, askBase, bidQuote *big.Int) *chPool {
	p := &chPool{bandWad: new(big.Int)}
	huge := new(big.Int).Lsh(chOne, 160)
	outDepth := [2]*big.Int{askBase, bidQuote}
	dec := [2]uint8{l.bd, l.qd}
	r := rng.Intn(20)
	switch {
	case l.extreme || r < 6:
		p.ideal, l.class = true, "ideal"
	case r < 15:
		l.class = "launch"
	default:
		l.class = "fee"
		p.spreadBp = 500 + uint64(rng.Intn(2501))
		if rng.Intn(2) == 0 {
			p.rate = uint64(rng.Intn(10001))
		}
		switch rng.Intn(3) {
		case 1:
			p.bandWad = new(big.Int).Set(chMinBand)
		case 2:
			p.bandWad = chBetween(rng, chWad, 1, 100)
		}
	}
	for y := range 2 {
		p.unit[y] = big.NewInt(1)
		if dec[y] > 6 {
			p.unit[y] = chPow10(uint64(dec[y]) - 6)
		}
		p.held[y], p.owed[y], p.capN[y] = new(big.Int).Set(huge), new(big.Int), new(big.Int)
		p.level[y] = new(big.Int).Set(huge)
		if p.ideal || outDepth[y].Sign() == 0 {
			continue
		}
		if rng.Intn(5) < 2 {
			p.held[y] = new(big.Int).Add(chBetween(rng, outDepth[y], 3, 110), chOne)
		}
		if rng.Intn(5) < 2 {
			p.level[y] = new(big.Int).Quo(chBetween(rng, outDepth[y], 3, 110), p.unit[y])
		}
		if rng.Intn(10) < 3 {
			p.capN[y] = new(big.Int).Add(chBetween(rng, outDepth[y], 10, 200), chOne)
		}
	}
	if l.class == "fee" && rng.Intn(3) == 0 && p.held[1].Cmp(huge) < 0 {
		p.owed[1] = chBetween(rng, p.held[1], 0, 20)
	}
	return p
}

// ---------------------------------------------------------------- the tracker's refresh, rebuilt

var chTokens = [2]string{simWETH, simUSDC}

func chEntity(l *chListing) entity.Pool {
	se, _ := json.Marshal(StaticExtra{Front: simFront, Listing: 1, Custody: simCustody, Door: simDoor, Mode: modeFront})
	d0, d1 := l.bd, l.qd
	if !l.base0 {
		d0, d1 = l.qd, l.bd
	}
	return entity.Pool{Address: simDoor, Exchange: DexType, Type: DexType, BlockNumber: 77, Timestamp: 1_764_000_000,
		Tokens: []*entity.PoolToken{{Address: chTokens[0], Decimals: d0, Swappable: true},
			{Address: chTokens[1], Decimals: d1, Swappable: true}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: string(se)}
}

// chRefresh is GetNewPoolState's grid and ladders (pool_tracker.go :354-383, :423-449) on the model's answers: the
// sizes of probeSizes, the side's top, and under layout the rung boundaries, the rung middles and the minimum-fill
// sizes; the door's quote at every size in every block; the least per size, nil where any block refused; the run;
// and, under layout, what a later fill is priced by.
func chRefresh(t *PoolTracker, l *chListing, p entity.Pool, blocks []chBlock, layout bool, st *chTally) (entity.Pool,
	Extra) {
	v := zeroView()
	v.PairWord, v.Word = l.pair, l.word
	v.Depth = [4]*big.Int{l.bid.w0, l.bid.w1, l.ask.w0, l.ask.w1}
	r := rungRead{board: [2]*big.Int{l.bid.board, l.ask.board}, controls: l.ctl, settings: l.s1, ok: true}
	reserves := l.reserves()
	var last Extra
	if layout && p.Extra != "" {
		_ = json.Unmarshal([]byte(p.Extra), &last)
	}
	extra := Extra{TipCapWei: 1e8, ValidUntil: 1 << 60, BaseToken1: !l.base0}
	var points [2][]*big.Int
	var samples [2][][]*big.Int
	for dir := range 2 {
		s := l.side(dir)
		points[dir] = t.probeSizes(p, dir, reserves[1-dir])
		if top, _ := s.fillable(blocks[0]); top.Sign() > 0 && top.Cmp(maxPointInput) < 0 {
			points[dir] = withSize(points[dir], top)
		}
		if layout {
			points[dir] = withEdges(points[dir], rungEdges(v, dir, r))
			var first Point
			if ld := last.ladders()[dir]; len(ld) > 0 {
				first = ld[0]
			}
			for _, size := range minSizes(v, dir, r, first) {
				if size.Sign() > 0 && size.Cmp(maxPointInput) < 0 {
					points[dir] = withSize(points[dir], size)
				}
			}
		}
		var outputs []*big.Int
		for bi, blk := range blocks {
			sample := make([]*big.Int, len(points[dir]))
			for i, size := range points[dir] {
				sample[i] = s.quote(size, blk)
			}
			if bi == 0 {
				outputs = slices.Clone(sample)
			} else {
				for i, out := range sample {
					if cur := outputs[i]; cur != nil && (out == nil || out.Cmp(cur) < 0) {
						outputs[i] = out
					}
				}
			}
			samples[dir] = append(samples[dir], sample)
		}
		ok := make([]bool, len(outputs))
		for i, o := range outputs {
			ok[i] = o != nil
		}
		extra.Ladders[dir], _ = collectClassedRun(points[dir], outputs, ok, nil)
		st.grid = max(st.grid, len(points[dir]))
	}
	// a listing whose pool takes a fee in the token it pays out is named in its front's config (OneFillListings): it
	// takes one fill a route
	if layout && l.class != "fee" {
		extra.Repeat = repeatOf(v, r, points, extra.Ladders, samples)
	}
	b, _ := json.Marshal(extra)
	p.Extra = string(b)
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	for dir, ld := range extra.Ladders {
		if len(ld) == 0 {
			p.Reserves[1-dir] = "0"
		}
	}
	return p, extra
}

// ---------------------------------------------------------------- the tally

const (
	chFirstPoint = iota
	chFirstChord
	chLaterOwn
	chLaterSteady
	chFirstPointOne
	chFirstChordOne
	chCats
)

var chCatName = [chCats]string{"first fill, at a point", "first fill, between points", "later fill, own ladder",
	"later fill, steady ladder", "first fill, at a point, one block sampled", "first fill, between points, one block"}

var chAbsEdges = []int64{0, 1, 2, 5, 10, 100, 10000}
var chPpmEdges = []int64{0, 1, 10, 100, 1000, 10000, 100000}

type chTally struct {
	listings, sides, sidesLater, points int
	kinds                               [3]int
	quoted, paid, over                  [chCats]int
	abs, ppm                            [chCats][8]int
	maxAbs                              [chCats]*big.Int
	worstOver                           *big.Int
	worstNote                           string
	reverts                             map[string]int
	refusals                            map[string]int
	examples                            map[string]string
	kind0Wrong, cloneWrong, simErr      int
	skipped, grid                       int
	outBits, inBits, wide               int
	maxSizes                            int
}

func chNewTally() *chTally {
	t := &chTally{worstOver: new(big.Int), reverts: map[string]int{}, refusals: map[string]int{},
		examples: map[string]string{}}
	for i := range t.maxAbs {
		t.maxAbs[i] = new(big.Int)
	}
	return t
}

func (t *chTally) add(o *chTally) {
	t.listings += o.listings
	t.sides += o.sides
	t.sidesLater += o.sidesLater
	t.points += o.points
	t.kind0Wrong += o.kind0Wrong
	t.cloneWrong += o.cloneWrong
	t.simErr += o.simErr
	t.skipped += o.skipped
	t.grid = max(t.grid, o.grid)
	t.outBits, t.inBits, t.wide = max(t.outBits, o.outBits), max(t.inBits, o.inBits), t.wide+o.wide
	t.maxSizes = max(t.maxSizes, o.maxSizes)
	for i := range t.kinds {
		t.kinds[i] += o.kinds[i]
	}
	for c := range chCats {
		t.quoted[c] += o.quoted[c]
		t.paid[c] += o.paid[c]
		t.over[c] += o.over[c]
		for i := range t.abs[c] {
			t.abs[c][i] += o.abs[c][i]
			t.ppm[c][i] += o.ppm[c][i]
		}
		if o.maxAbs[c].Cmp(t.maxAbs[c]) > 0 {
			t.maxAbs[c] = o.maxAbs[c]
		}
	}
	if o.worstOver.Cmp(t.worstOver) > 0 {
		t.worstOver, t.worstNote = o.worstOver, o.worstNote
	}
	for k, n := range o.reverts {
		t.reverts[k] += n
	}
	for k, n := range o.refusals {
		t.refusals[k] += n
	}
	for k, n := range o.examples {
		if _, ok := t.examples[k]; !ok {
			t.examples[k] = n
		}
	}
}

func chBucket(edges []int64, v *big.Int) int {
	for i, e := range edges {
		if v.Cmp(big.NewInt(e)) <= 0 {
			return i
		}
	}
	return len(edges)
}

// ---------------------------------------------------------------- a route

func chCalc(sim *PoolSimulator, dir int, amount *big.Int, limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	return calc(sim, chTokens[dir], chTokens[1-dir], amount, limit)
}

func chApply(sim *PoolSimulator, dir int, amount *big.Int, res *pool.CalcAmountOutResult, limit pool.SwapLimit) {
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: chTokens[dir], Amount: amount},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
		SwapLimit:      limit,
	})
}

// chSize is a size for a direction's next fill: used is the input the route's fills of it took, rounded as the front
// rounds it.
func chSize(rng *rand.Rand, pts []Point, used, unit *big.Int) *big.Int {
	first, top := pts[0][0].ToBig(), pts[len(pts)-1][0].ToBig()
	left := new(big.Int).Sub(top, used)
	if left.Sign() <= 0 {
		return new(big.Int).Set(first)
	}
	switch rng.Intn(12) {
	case 0, 1, 2:
		to := pts[rng.Intn(len(pts))][0].ToBig()
		if a := new(big.Int).Sub(to, used); a.Sign() > 0 {
			a.Add(a, big.NewInt(int64(rng.Intn(3)-1)))
			if a.Sign() > 0 {
				return a
			}
		}
	case 3, 4:
		return left
	case 5:
		return new(big.Int).Set(first)
	case 6:
		return new(big.Int).Add(first, chRandBelow(rng, new(big.Int).Add(first, chOne)))
	case 7:
		// a whole number of board units
		n := new(big.Int).Quo(left, unit)
		if n.Sign() > 0 {
			a := new(big.Int).Add(chRandBelow(rng, n), chOne)
			return a.Mul(a, unit)
		}
	case 8:
		if left.Cmp(chOne) > 0 {
			return new(big.Int).Sub(left, chOne)
		}
	}
	return new(big.Int).Add(chRandBelow(rng, left), chOne)
}

func chFrontCause(c string) bool {
	switch c {
	case "OutflowCap", "OutputAboveCeiling", "NotionalCap(in)", "NotionalCap(out)":
		return false
	}
	return true
}

// chAhead is whether the standing word may refill the side at a later fill of the route and not at the first: the
// controls let a word refill, the word asks it of the side, and it acknowledges a sequence the route's own fills
// reach (PropAMMWords.refills :78-82).
func chAhead(s *chSide) bool {
	on, ack := s.word.Bit(233) == 1, chU(s.word, 245, 11)
	if s.bid {
		on, ack = s.word.Bit(232) == 1, chU(s.word, 234, 11)
	}
	ahead := (ack - chU(s.board, 236, 11)) & 0x7ff
	return s.ctl.Bit(66) == 1 && on && ahead >= 1 && ahead <= 8
}

// chRoute asks the simulator for a route of two to four fills, most of one direction, some of the other between
// them, and runs every fill it quotes on the model in one block. It tallies each fill against the chain.
func chRoute(rng *rand.Rand, l *chListing, p entity.Pool, extra Extra, blocks []chBlock, st *chTally, tag string,
	main int) {
	if len(extra.Ladders[main]) == 0 {
		return
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	if err != nil {
		st.simErr++
		return
	}
	var limit pool.SwapLimit
	if rng.Intn(2) == 0 {
		limit = swaplimit.NewInventory(DexType, sim.CalculateLimit())
	}
	chain := l.chain()
	blk := blocks[rng.Intn(len(blocks))]
	var kind [2]uint8
	if extra.Repeat != nil {
		kind = extra.Repeat.Kind
	}
	st.sides++
	st.kinds[kind[main]]++
	st.points += len(extra.Ladders[main])
	var unit, used [2]*big.Int
	var ahead, refilledLater [2]bool
	var count [2]int
	for d := range 2 {
		uexp := chU(l.pair, 120, 8)
		if chain.side(d).bid {
			uexp = chU(l.pair, 112, 8)
		}
		unit[d], used[d] = chPow10(uexp), new(big.Int)
		ahead[d] = chAhead(chain.side(d))
	}
	if ahead[main] {
		st.skipped++
	}
	var trail []string
	later := false
	fills := 2 + rng.Intn(3)
	for f := 0; f < fills; f++ {
		dir := main
		if f > 0 && rng.Intn(4) == 0 && len(extra.Ladders[1-main]) > 0 {
			dir = 1 - main
		}
		pts := extra.Ladders[dir]
		side := chain.side(dir)
		var amount *big.Int
		var res *pool.CalcAmountOutResult
		for try := 0; try < 4; try++ {
			amount = chSize(rng, pts, used[dir], unit[dir])
			if count[dir] == 0 && amount.Cmp(pts[0][0].ToBig()) < 0 {
				amount = pts[0][0].ToBig()
			}
			res, err = chCalc(sim, dir, amount, limit)
			if err == nil {
				break
			}
			if count[dir] > 0 {
				st.refusals[fmt.Sprintf("kind %d: %s", kind[dir], err)]++
				if kind[dir] == 0 && !errors.Is(err, ErrSecondFill) {
					st.kind0Wrong++
				}
			}
		}
		if err != nil {
			if dir != main {
				continue
			}
			return
		}
		cat := chFirstChord
		switch {
		case count[dir] > 0 && kind[dir] == repeatOwn:
			cat = chLaterOwn
		case count[dir] > 0:
			cat = chLaterSteady
		default:
			for _, pt := range pts {
				if pt[0].ToBig().Cmp(amount) == 0 {
					cat = chFirstPoint
				}
			}
			if len(blocks) == 1 {
				cat += chFirstPointOne - chFirstPoint
			}
		}
		if count[dir] > 0 && dir == main && !later {
			later = true
			st.sidesLater++
		}
		quoted := res.TokenAmountOut.Amount
		st.quoted[cat]++
		trail = append(trail, fmt.Sprintf("dir %d %s->%s", dir, amount, quoted))

		// a clone taken before the fill keeps the state before it
		var before *PoolSimulator
		if limit == nil && rng.Intn(4) == 0 {
			before = sim
			sim = sim.CloneState().(*PoolSimulator)
		}

		// whether the front refills the side at this fill, a later one
		if count[dir] > 0 && side.ctl.Bit(66) == 1 && side.refills(side.board) {
			refilledLater[dir] = true
		}
		refilled := refilledLater[dir]
		out, cause := side.swap(amount, blk)
		if cause != "" {
			who := "pool"
			if chFrontCause(cause) {
				who = "front"
			}
			if refilled {
				cause += " (the word refilled the side at a later fill of the route)"
			}
			key := fmt.Sprintf("%s|%s|%s|%s", chCatName[cat], l.class, who, cause)
			st.reverts[key]++
			if _, ok := st.examples[key]; !ok {
				st.examples[key] = fmt.Sprintf("%s bid %v dec %d/%d c %d kind %d fills [%s] ladder top %s->%s "+
					"pool held %s/%s owed %s level %s/%s unit %s/%s cap %s/%s rate %d spread %d", tag, side.bid,
					l.bd, l.qd, blk.c, kind[dir], strings.Join(trail, "; "), pts[len(pts)-1][0], pts[len(pts)-1][1],
					l.pool.held[0], l.pool.held[1], l.pool.owed[1], l.pool.level[0], l.pool.level[1],
					l.pool.unit[0], l.pool.unit[1], l.pool.capN[0], l.pool.capN[1], l.pool.rate, l.pool.spreadBp)
			}
			return
		}
		st.paid[cat]++
		st.outBits, st.inBits = max(st.outBits, out.BitLen()), max(st.inBits, amount.BitLen())
		if out.BitLen() > 100 {
			st.wide++
		}
		d := new(big.Int).Sub(out, quoted)
		if d.Sign() < 0 {
			st.over[cat]++
			d.Neg(d)
			if d.Cmp(st.worstOver) > 0 {
				st.worstOver = d
				st.worstNote = fmt.Sprintf("%s bid %v dec %d/%d c %d kind %d class %s fills [%s] chain paid %s",
					tag, side.bid, l.bd, l.qd, blk.c, kind[dir], l.class, strings.Join(trail, "; "), out)
			}
		} else {
			st.abs[cat][chBucket(chAbsEdges, d)]++
			if d.Cmp(st.maxAbs[cat]) > 0 {
				st.maxAbs[cat] = d
			}
			rel := new(big.Int).Mul(d, chPPM)
			rel.Add(rel, new(big.Int).Sub(out, chOne)).Quo(rel, out)
			st.ppm[cat][chBucket(chPpmEdges, rel)]++
		}
		chApply(sim, dir, amount, res, limit)
		if before != nil {
			again, err := chCalc(before, dir, amount, nil)
			if err != nil || again.TokenAmountOut.Amount.Cmp(quoted) != 0 {
				st.cloneWrong++
			}
		}
		u := new(big.Int).Add(amount, unit[dir])
		u.Sub(u, chOne).Quo(u, unit[dir]).Mul(u, unit[dir])
		used[dir].Add(used[dir], u)
		count[dir]++
	}
}

// chLayout is whether the refresh reads the front's words by their layout: CHAIN_LAYOUT=0 is a front whose
// implementation the config does not list, or one that does not answer extSloads (pool_tracker.go :354).
var chLayout = os.Getenv("CHAIN_LAYOUT") != "0"

// chExtreme makes every listing one whose prices put outputs near 2^128: CHAIN_EXTREME=1.
var chExtreme = os.Getenv("CHAIN_EXTREME") == "1"

func chEnvInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

func chOneListing(seed int64, st *chTally) {
	rng := rand.New(rand.NewSource(seed))
	l, blocks := chGen(rng)
	t := &PoolTracker{}
	st.listings++
	p := chEntity(l)
	refreshes := 1 + rng.Intn(3)
	var extra Extra
	for i := 0; i < refreshes; i++ {
		p, extra = chRefresh(t, l, p, blocks, chLayout, st)
	}
	for dir := range 2 {
		st.maxSizes = max(st.maxSizes, len(extra.Ladders[dir]))
	}
	for r := 0; r < 2; r++ {
		chRoute(rng, l, p, extra, blocks, st, fmt.Sprintf("seed %d refresh %d route %d", seed, refreshes, r), r)
	}
}

func TestChainProperty(t *testing.T) {
	listings := chEnvInt("CHAIN_LISTINGS", 4000)
	workers := chEnvInt("CHAIN_WORKERS", 4)
	first := int64(chEnvInt("CHAIN_FIRST", 1))
	total := chNewTally()
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			st := chNewTally()
			for i := w; i < listings; i += workers {
				chOneListing(first+int64(i), st)
			}
			mu.Lock()
			total.add(st)
			mu.Unlock()
		}(w)
	}
	wg.Wait()

	t.Logf("listings %d, sides (a listing's direction with a ladder, one route each) %d, of them with a later fill "+
		"quoted %d; ladder points %d (longest %d)", total.listings, total.sides, total.sidesLater, total.points,
		total.maxSizes)
	t.Logf("sides by what a later fill is priced by: none %d, steady %d, own %d", total.kinds[0], total.kinds[1],
		total.kinds[2])
	for c := range chCats {
		t.Logf("%-42s quoted %8d  paid %8d  over-quoted %d  largest shortfall %s", chCatName[c], total.quoted[c],
			total.paid[c], total.over[c], total.maxAbs[c])
		t.Logf("    shortfall, base units of the output  (=0, 1, 2, <=5, <=10, <=100, <=1e4, more): %v", total.abs[c])
		t.Logf("    shortfall, ppm of the fill rounded up (=0, <=1, <=10, <=100, <=1e3, <=1e4, <=1e5, more): %v",
			total.ppm[c])
	}
	keys := make([]string, 0, len(total.reverts))
	for k := range total.reverts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("QUOTED, REFUSED ON CHAIN  %-90s %d", k, total.reverts[k])
		if e, ok := total.examples[k]; ok {
			t.Logf("    e.g. %s", e)
		}
	}
	keys = keys[:0]
	for k := range total.refusals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("later fill not quoted     %-90s %d", k, total.refusals[k])
	}
	t.Logf("worst over-quote where the chain paid: %s  %s", total.worstOver, total.worstNote)
	t.Logf("most sizes a refresh asked of one direction: %d; routes whose side's word may refill it at a later fill: %d",
		total.grid, total.skipped)
	t.Logf("widest fill paid: input %d bits, output %d bits; fills paid over 2^100 of output: %d", total.inBits,
		total.outBits, total.wide)
	t.Logf("a direction that proves nothing answered other than ErrSecondFill: %d; a clone moved its source: %d; "+
		"states the simulator refused: %d", total.kind0Wrong, total.cloneWrong, total.simErr)
	if total.worstOver.Sign() != 0 {
		t.Errorf("the simulator quoted above the chain")
	}
	for k, n := range total.reverts {
		t.Errorf("a fill the simulator quoted was refused on chain: %s x%d", k, n)
	}
	assert.Empty(t, total.reverts, "no quoted fill is refused, by the front or by the pool")
	assert.Greater(t, total.sidesLater, 100, "the run quotes later fills")
	if total.kind0Wrong != 0 || total.cloneWrong != 0 {
		t.Errorf("state errors")
	}
}
