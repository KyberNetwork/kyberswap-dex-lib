package efficientprop

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const bookPPM = 1_000_000

var (
	bookBidScale = new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)                 // 1e36
	bookAskScale = new(big.Int).Lsh(big.NewInt(1), 96)                                   // ASK_SCALE
	bookAskNum   = new(big.Int).Mul(bookBidScale, bookAskScale)                          // ASK_NUMERATOR
	bookCap      = new(big.Int).Lsh(big.NewInt(1), 128)                                  // the front's bound on either leg
	bookNoMemory = uint64(1)<<48 - 1                                                     // a depth tag no test post carries
	bookExpires  = uint64(4_000_000_000)                                                 // a depth post's end, s
	bookHuge     = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1)) // more than any side holds
)

// fakeBook is one side of a listing as the front stores and fills it: its depth post and board word packed as the
// front packs them (PropAMMWords), and a walk that reads those words as PropAMMFront._walk does, step for step. It
// stands for the contract in the rung tests: the tracker's reading of the words (rungEdges) is checked against a
// walk that never calls it.
type fakeBook struct {
	bid              bool
	sizeExp, unitExp uint
	// each rung's cumulative size (16 bits, strictly rising) and offset (ppm, never falling, the first 0)
	sizes, offsets []uint64
	// price is the maker's level price, the output per input times 1e36; lim, where set, the fence's clip of it
	price, lim *big.Int
	// nonce is the depth post's; tag and filled the board word's depth tag and fill memory (board units), seq its
	// fill sequence
	nonce, tag, filled, seq uint64
	// minFill is the least the side fills, on its quote leg as the front counts it: the output of a bid, the input of
	// an ask; outBound, where set, the most it pays out (its pool's room, its desk's bucket), which a fill takes from
	minFill, outBound *big.Int
	// discount is the front's discount at the block, ppm, added to every rung's offset alike (PropAMMFront._discount)
	discount uint64
}

// at is the side in a block where the front's discount is ppm more.
func (b *fakeBook) at(ppm uint64) *fakeBook {
	c := *b
	c.discount += ppm
	return &c
}

// memory is the fill memory the side fills from, board units: none under another post's tag.
func (b *fakeBook) memory() uint64 {
	if b.tag != b.nonce {
		return 0
	}
	return b.filled
}

// fill is a fill of amountIn, all or nothing as quote, which moves the side as the front's does: the fill memory by
// the input rounded up to the board unit, under the post's tag, the sequence by one, and the output bound by what it
// paid (PropAMMFront._evaluate, the side's new board state).
func (b *fakeBook) fill(amountIn *big.Int) (*big.Int, error) {
	out, err := b.quote(amountIn)
	if err != nil {
		return nil, err
	}
	units := new(big.Int).Add(amountIn, b.unit())
	units.Sub(units, big.NewInt(1)).Quo(units, b.unit())
	b.filled, b.tag, b.seq = b.memory()+units.Uint64(), b.nonce, b.seq+1
	if b.outBound != nil {
		b.outBound = new(big.Int).Sub(b.outBound, out)
	}
	return out, nil
}

func shl(v uint64, n uint) *big.Int { return new(big.Int).Lsh(new(big.Int).SetUint64(v), n) }

// amount24 is the front's 24-bit amount at or under v, an 18-bit mantissa times ten to a 6-bit exponent, and the
// amount it stands for (PropAMMLaneLib.amount24).
func amount24(v *big.Int) (field uint64, amount *big.Int) {
	m, e := new(big.Int).Set(v), uint64(0)
	for m.BitLen() > 18 {
		m.Quo(m, big.NewInt(10))
		e++
	}
	return m.Uint64() | e<<18, m.Mul(m, pow10(uint(e)))
}

func pow10(e uint) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil) }

// depth is the side's two depth words (PropAMMWords: depth).
func (b *fakeBook) depth() (d0, d1 *big.Int) {
	d0, d1 = big.NewInt(1), new(big.Int)
	d0.Or(d0, shl(b.nonce, 12)).Or(d0, shl(bookExpires, 60)).Or(d0, shl(uint64(b.sizeExp), 112))
	for j := range b.sizes {
		rung := b.sizes[j] | b.offsets[j]<<16
		if j < 3 {
			d0.Or(d0, shl(rung, 120+36*uint(j)))
		} else {
			d1.Or(d1, shl(rung, 36*uint(j-3)))
		}
	}
	if len(b.sizes) > 3 {
		d0.SetBit(d0, 118, 1)
	}
	return d0, d1
}

// board is the side's board word (PropAMMWords.board): its depth tag, fill memory and fill sequence.
func (b *fakeBook) board() *big.Int {
	w := big.NewInt(1)
	return w.Or(w, shl(b.tag, 4)).Or(w, shl(b.filled, 52)).Or(w, shl(b.seq, 236))
}

// unit is the side's board unit, in base units of its input token.
func (b *fakeBook) unit() *big.Int { return pow10(b.unitExp) }

// level is a rung's price as the walk takes it, at offset d: the maker's price less the offset, clipped to the fence
// (PropAMMFront._unclamped and the walk's clamp), the output per input times 1e36 for a bid and times 2^96 for an ask.
func (b *fakeBook) level(d uint64) *big.Int {
	ppm := big.NewInt(bookPPM)
	if b.bid {
		pu := new(big.Int).Mul(b.price, big.NewInt(int64(bookPPM-d)))
		pu.Quo(pu, ppm)
		if b.lim != nil && b.lim.Cmp(pu) < 0 {
			pu.Set(b.lim)
		}
		return pu
	}
	den := big.NewInt(int64(bookPPM - d))
	px := new(big.Int).Mul(b.price, ppm)
	px.Add(px, den).Sub(px, big.NewInt(1)).Quo(px, den)
	if b.lim != nil && b.lim.Cmp(px) > 0 {
		px.Set(b.lim)
	}
	if px.Sign() == 0 {
		return new(big.Int)
	}
	return new(big.Int).Quo(bookAskNum, px)
}

// scale is what the walk divides its scaled output by, once.
func (b *fakeBook) scale() *big.Int {
	if b.bid {
		return bookBidScale
	}
	return bookAskScale
}

// walk is PropAMMFront._walk on the packed words: the input it fills, x, of amountIn and the scaled output, n.
func (b *fakeBook) walk(amountIn *big.Int) (x, n *big.Int) {
	x, n = new(big.Int), new(big.Int)
	if amountIn.Sign() <= 0 {
		return x, n
	}
	w0, w1 := b.depth()
	bd := b.board()
	unit := b.unit()
	scale := new(big.Int).Mul(pow10(uint(bits(w0, 112, 6))), unit)
	filled := uint64(0)
	if bits(bd, 4, 48) == bits(w0, 12, 48) {
		filled = bits(bd, 52, 40)
	}
	cursor := new(big.Int).Mul(new(big.Int).SetUint64(filled), unit)
	outBound := bookCap
	if b.outBound != nil && b.outBound.Cmp(bookCap) < 0 {
		outBound = b.outBound
	}
	limit := new(big.Int).Add(outBound, big.NewInt(1))
	limit.Mul(limit, b.scale()).Sub(limit, big.NewInt(1))
	prev := new(big.Int)
	for j := range uint(10) {
		if j == 3 && w0.Bit(118) == 0 {
			break
		}
		rg := bits(w0, 120+36*j, 36)
		if j >= 3 {
			rg = bits(w1, 36*(j-3), 36)
		}
		size := rg & 0xffff
		if size == 0 {
			break
		}
		h := new(big.Int).Mul(new(big.Int).SetUint64(size), scale)
		if h.Cmp(cursor) <= 0 {
			prev = h
			continue
		}
		d := rg>>16 + b.discount
		if d >= bookPPM {
			break
		}
		q := b.level(d)
		if q.Sign() == 0 {
			break
		}
		from := prev
		if cursor.Cmp(prev) > 0 {
			from = cursor
		}
		t := new(big.Int).Sub(h, from)
		if left := new(big.Int).Sub(amountIn, x); t.Cmp(left) > 0 {
			t = left
		}
		room := new(big.Int).Sub(limit, n)
		if most := new(big.Int).Quo(room, q); t.Cmp(most) > 0 {
			x.Add(x, most)
			n.Add(n, most.Mul(most, q))
			break
		}
		n.Add(n, new(big.Int).Mul(t, q))
		x.Add(x, t)
		if x.Cmp(amountIn) == 0 {
			break
		}
		prev = h
	}
	return x, n
}

// quote is the door's quote of the side: all of amountIn or nothing, the output floored once, its quote leg no less
// than the side's minimum fill (PropAMMFront._evaluate and _quoteTail).
func (b *fakeBook) quote(amountIn *big.Int) (*big.Int, error) {
	x, n := b.walk(amountIn)
	out := n.Quo(n, b.scale())
	if amountIn.Sign() <= 0 || x.Cmp(amountIn) != 0 || out.Sign() == 0 {
		return nil, errNoFill
	}
	leg := x
	if b.bid {
		leg = out
	}
	if b.minFill != nil && leg.Cmp(b.minFill) < 0 {
		return nil, errBelowMinimum
	}
	return out, nil
}

// top is the most the side fills at once (the door's quoteFillable of the front's most): 0 where it fills nothing.
func (b *fakeBook) top() *big.Int {
	x, _ := b.walk(bookHuge)
	if _, err := b.quote(x); err != nil {
		return new(big.Int)
	}
	return x
}

// seen is the side as the tracker's head read finds it: the direction it is of a pair whose base is token 0, a
// doorView holding its depth words and board exponent, and the board word beside it.
func (b *fakeBook) seen() (dir int, v DoorView, r rungRead) {
	v = zeroView()
	d0, d1 := b.depth()
	r = rungRead{board: [2]*big.Int{new(big.Int), new(big.Int)}, controls: new(big.Int), settings: new(big.Int),
		ok: true}
	if b.minFill != nil {
		field, _ := amount24(b.minFill)
		r.settings = shl(field, 160)
	}
	v.PairWord = big.NewInt(1 << 8)
	if b.bid {
		v.PairWord.Or(v.PairWord, shl(uint64(b.unitExp), 112))
		v.Depth[0], v.Depth[1], r.board[0] = d0, d1, b.board()
		return 0, v, r
	}
	v.PairWord.Or(v.PairWord, shl(uint64(b.unitExp), 120))
	v.Depth[2], v.Depth[3], r.board[1] = d0, d1, b.board()
	return 1, v, r
}

// edges is the side's rung boundaries as the tracker reads them.
func (b *fakeBook) edges() []*big.Int {
	dir, v, r := b.seen()
	return rungEdges(v, dir, r)
}

// randomBook is a seeded side of 1 to 10 rungs: a WETH bid for USDC at about 3,000, or the ask for it. Its rungs
// may share a price (an equal offset, or the fence clipping several to one), its fill memory may sit inside a rung,
// exactly on a boundary, past whole rungs or past them all, or belong to an older post, and its output may be capped.
func randomBook(rng *rand.Rand, bid bool) *fakeBook {
	b := &fakeBook{bid: bid, sizeExp: uint(rng.Intn(3)), unitExp: uint(rng.Intn(4)), nonce: 1 + uint64(rng.Int63n(1<<40))}
	if bid {
		b.unitExp += 11
	}
	n := 1 + rng.Intn(10)
	if rng.Intn(6) == 0 {
		n = 1
	}
	size, off := uint64(0), uint64(0)
	for j := 0; j < n && size < 60_000; j++ {
		size += 1 + uint64(rng.Intn(5_000))
		if j > 0 && rng.Intn(3) > 0 {
			off += uint64(rng.Intn(600))
		}
		b.sizes, b.offsets = append(b.sizes, size), append(b.offsets, off)
	}
	// 3,000 USDC a WETH, in base units of the output per base unit of the input, times 1e36
	b.price = new(big.Int).Mul(big.NewInt(2_000+rng.Int63n(2_000)), pow10(24))
	if rng.Intn(3) == 0 {
		// the fence clips every rung before one of them to that one's price
		k := rng.Intn(len(b.sizes))
		ppm, d := big.NewInt(bookPPM), int64(b.offsets[k])
		if bid {
			b.lim = new(big.Int).Mul(b.price, big.NewInt(bookPPM-d))
			b.lim.Quo(b.lim, ppm)
		} else {
			b.lim = new(big.Int).Mul(b.price, ppm)
			b.lim.Quo(b.lim, big.NewInt(bookPPM-d))
		}
	}
	b.tag = b.nonce
	last := b.sizes[len(b.sizes)-1] * uint64(pow10(b.sizeExp).Int64())
	switch rng.Intn(8) {
	case 0, 1, 2: // inside a rung, or past some
		b.filled = uint64(rng.Int63n(int64(last)))
	case 3: // exactly on a rung's boundary: the rung is used up, the next untouched
		b.filled = b.sizes[rng.Intn(len(b.sizes))] * uint64(pow10(b.sizeExp).Int64())
	case 4: // an older post's memory: the side fills from its start
		b.tag, b.filled = bookNoMemory, uint64(rng.Int63n(int64(last)))
	case 5:
		if rng.Intn(4) == 0 { // past every rung: nothing left
			b.filled = last + uint64(rng.Intn(3))
		}
	}
	whole, _ := b.walk(bookHuge)
	_, n256 := b.walk(whole)
	full := n256.Quo(n256, b.scale())
	if full.Sign() > 0 {
		leg := whole
		if bid {
			leg = full
		}
		// an amount the listing's settings can hold
		_, b.minFill = amount24(new(big.Int).Rand(rng, new(big.Int).Add(new(big.Int).Quo(leg, big.NewInt(50)),
			big.NewInt(1))))
		if rng.Intn(5) == 0 {
			// the pool's room ends inside the side
			b.outBound = new(big.Int).Rand(rng, full)
		}
	}
	return b
}

// bookPool is a WETH/USDC pool as the tracker's sizing reads it, its last state's ladder for direction dir prev (none:
// a first refresh).
func bookPool(dir int, prev []Point) entity.Pool {
	p := entity.Pool{
		Tokens:   []*entity.PoolToken{{Address: simWETH, Decimals: 18}, {Address: simUSDC, Decimals: 6}},
		Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000"},
	}
	if len(prev) > 0 {
		var e Extra
		e.Ladders[dir] = prev
		extra, _ := json.Marshal(e)
		p.Extra = string(extra)
	}
	return p
}

// bookLadder is the ladder a refresh keeps of a side asked at sizes: the door's own answers, as the tracker collects
// them.
func bookLadder(b *fakeBook, sizes []*big.Int) []Point {
	outputs, ok := make([]*big.Int, len(sizes)), make([]bool, len(sizes))
	for i, size := range sizes {
		if out, err := b.quote(size); err == nil {
			outputs[i], ok[i] = out, true
		}
	}
	run, _ := collectClassedRun(sizes, outputs, ok, nil)
	return run
}

// bookGrid is the sizes the tracker asks a side at in a refresh that follows a state with ladder prev: its grid and
// the side's top, and with edges, the side's rung boundaries and the sizes at its minimum fill.
func bookGrid(b *fakeBook, prev []Point, edges bool) []*big.Int {
	dir := 1
	if b.bid {
		dir = 0
	}
	p := bookPool(dir, prev)
	reserve, _ := new(big.Int).SetString(p.Reserves[1-dir], 10)
	sizes := (&PoolTracker{}).probeSizes(p, dir, reserve)
	if top := b.top(); top.Sign() > 0 {
		sizes = withSize(sizes, top)
	}
	if edges {
		sizes = withEdges(sizes, b.edges())
		var first Point
		if len(prev) > 0 {
			first = prev[0]
		}
		_, v, r := b.seen()
		for _, size := range minSizes(v, dir, r, first) {
			sizes = withSize(sizes, size)
		}
	}
	return sizes
}

// refreshed is a side's ladder after two refreshes, the second sized from the first, with or without the rung
// boundaries.
func refreshed(b *fakeBook, edges bool) []Point {
	return bookLadder(b, bookGrid(b, bookLadder(b, bookGrid(b, nil, edges)), edges))
}

// bookSim is a simulator whose ladder for the book's direction is points.
func bookSim(t *testing.T, b *fakeBook, points []Point) (sim *PoolSimulator, tokenIn, tokenOut string) {
	t.Helper()
	return bookSimOf(t, b, points, 0)
}

// bookSimOf is bookSim of a state that says what the ladder proves of a later fill (Repeat's kind; 0: nothing).
func bookSimOf(t *testing.T, b *fakeBook, points []Point, kind uint8) (sim *PoolSimulator, tokenIn, tokenOut string) {
	t.Helper()
	ladders, tokenIn, tokenOut := [2][]Point{points, nil}, simWETH, simUSDC
	if !b.bid {
		ladders, tokenIn, tokenOut = [2][]Point{nil, points}, simUSDC, simWETH
	}
	// a custody no side of these outgrows
	p := simPool(t, ladders, time.Now().Unix())
	p.Reserves = entity.PoolReserves{"1000000000000000000000000000000", "1000000000000000000000000000000"}
	if kind != 0 {
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		e.Repeat = &Repeat{}
		dir := 1
		if b.bid {
			dir = 0
		}
		e.Repeat.Unit[dir], e.Repeat.Kind[dir] = uint8(b.unitExp), kind
		extra, err := json.Marshal(e)
		require.NoError(t, err)
		p.Extra = string(extra)
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	return sim, tokenIn, tokenOut
}

// The boundaries the tracker reads are every size at which the walk changes rung: between two neighbours, and from
// nothing to the first, the walk's scaled output is one straight line of its input, exactly, and the last is the
// most the rungs fill.
func TestRungEdgesAreWhereTheWalkChangesRung(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	spans, empty := 0, 0
	for trial := range 600 {
		b := randomBook(rng, trial%2 == 0)
		b.outBound = nil
		edges := b.edges()
		if len(edges) == 0 {
			x, _ := b.walk(bookHuge)
			require.Zero(t, x.Sign(), "trial %d: no boundary read of a side that fills", trial)
			empty++
			continue
		}
		require.LessOrEqual(t, len(edges), maxRungs)
		x, _ := b.walk(bookHuge)
		require.Equal(t, edges[len(edges)-1], x, "trial %d: the last boundary is all the side fills", trial)
		lo, nLo := new(big.Int), new(big.Int)
		for _, hi := range edges {
			require.Positive(t, hi.Cmp(lo), "trial %d: boundaries rise", trial)
			xHi, nHi := b.walk(hi)
			require.Equal(t, hi, xHi)
			dx, dn := new(big.Int).Sub(hi, lo), new(big.Int).Sub(nHi, nLo)
			for range 6 {
				at := new(big.Int).Add(lo, new(big.Int).Rand(rng, dx))
				_, nAt := b.walk(at)
				// (n(at) - n(lo)) * dx == dn * (at - lo): one price over the whole span
				left := new(big.Int).Mul(new(big.Int).Sub(nAt, nLo), dx)
				right := new(big.Int).Mul(dn, new(big.Int).Sub(at, lo))
				require.Equal(t, right, left, "trial %d: a knee inside (%s, %s] at %s", trial, lo, hi, at)
			}
			lo, nLo = hi, nHi
			spans++
		}
	}
	assert.Greater(t, spans, 1500)
	assert.Greater(t, empty, 5, "sides with nothing left are among the trials")
}

// The fill memory decides where the boundaries are: it counts against the post it was filled from alone, and not past
// a refill, and words of another layout give no boundary at all.
func TestRungEdgesFollowTheFillMemory(t *testing.T) {
	book := func() *fakeBook {
		return &fakeBook{bid: true, sizeExp: 1, unitExp: 13, sizes: []uint64{100, 250, 400, 1000, 1500},
			offsets: []uint64{0, 10, 10, 50, 90}, price: new(big.Int).Mul(big.NewInt(3000), pow10(24)), nonce: 77, tag: 77}
	}
	e14 := func(v ...int64) []*big.Int {
		out := make([]*big.Int, len(v))
		for i, x := range v {
			out[i] = new(big.Int).Mul(big.NewInt(x), pow10(14))
		}
		return out
	}
	b := book()
	assert.Equal(t, e14(100, 250, 400, 1000, 1500), b.edges(), "an untouched side: every rung's cumulative size")

	b.filled = 1205 // board units: 120.5 sizes, inside the second rung
	want := []*big.Int{bi("12950000000000000"), bi("27950000000000000"), bi("87950000000000000"),
		bi("137950000000000000")}
	assert.Equal(t, want, b.edges(), "inside a rung: the rest of it, then the rungs after")

	b.filled = 2500
	assert.Equal(t, e14(150, 750, 1250), b.edges(), "on a boundary: the rung before is used up")

	b.tag = 76
	assert.Equal(t, e14(100, 250, 400, 1000, 1500), b.edges(), "another post's memory does not count")

	b.tag, b.filled = 77, 15_000
	assert.Empty(t, b.edges(), "nothing left")

	// a standing word that refills the side: the memory is void, as the front judges it
	b = book()
	b.filled, b.seq = 2500, 9
	dir, v, r := b.seen()
	refill := new(big.Int).SetBit(new(big.Int), 232, 1)
	refill.Or(refill, shl(9, 234)).Or(refill, shl(5_000, 72))
	v.Word = refill
	assert.Equal(t, e14(150, 750, 1250), rungEdges(v, dir, r), "the listing's controls do not let a word refill")
	r.controls = new(big.Int).SetBit(new(big.Int), ctlRefill, 1)
	assert.Equal(t, e14(100, 250, 400, 1000, 1500), rungEdges(v, dir, r), "a refill: from the start of the rungs")
	v.Word = new(big.Int).Or(new(big.Int).SetBit(new(big.Int), 232, 1), shl(8, 234))
	v.Word.Or(v.Word, shl(5_000, 72))
	assert.Equal(t, e14(150, 750, 1250), rungEdges(v, dir, r), "a word that acknowledges another fill refills nothing")
	v.Word = new(big.Int).Set(refill)
	r.board[0].Or(r.board[0], shl(5_000&0x1ff, 247))
	assert.Equal(t, e14(150, 750, 1250), rungEdges(v, dir, r), "the word that refilled the side last does not again")

	// the ask's words are the other pair of the view, its unit the pair word's other exponent
	a := book()
	a.bid, a.unitExp = false, 2
	dir, v, r = a.seen()
	require.Equal(t, 1, dir)
	assert.Equal(t, []*big.Int{big.NewInt(100_000), big.NewInt(250_000), big.NewInt(400_000), big.NewInt(1_000_000),
		big.NewInt(1_500_000)}, rungEdges(v, dir, r))
	assert.Empty(t, rungEdges(v, 0, r), "the other side has no post")

	// words this does not read give no boundary: the grid stays as it was
	b = book()
	dir, v, r = b.seen()
	v.Depth[0] = new(big.Int).Xor(v.Depth[0], big.NewInt(3)) // version 2
	assert.Empty(t, rungEdges(v, dir, r), "a depth word of another version")
	_, v, _ = b.seen()
	v.Depth[0].Or(v.Depth[0], shl(31, 112))
	assert.Empty(t, rungEdges(v, dir, r), "a size exponent the front never takes")
	_, v, _ = b.seen()
	v.PairWord.Or(v.PairWord, shl(17, 112))
	assert.Empty(t, rungEdges(v, dir, r), "a board exponent the front never sets")
	_, v, _ = b.seen()
	assert.Empty(t, rungEdges(v, dir, rungRead{}), "no storage read")
	v.Depth[0].SetBit(v.Depth[0], 118, 0)
	assert.Equal(t, e14(100, 250, 400), rungEdges(v, dir, r), "word 1 not present: three rungs")
}

// The chord between two neighbouring points of a ladder with a point on every rung boundary is the door's quote to a
// unit: at any size from the first point to the last, the simulator pays the door's output or one base unit of the
// output token less, never more. (Between two boundaries the scaled output F is one line; the points are floor(F), so
// the chord is under F by less than 1, and its floor under floor(F) by at most 1.) The grid without the boundaries,
// which chords across each knee, is measured beside it.
func TestRungEdgeChordIsWithinAUnitOfTheDoor(t *testing.T) {
	type tally struct {
		sizes, quoted int
		worst         *big.Int
		worstPpm      float64
		sumPpm        float64
	}
	measure := func(tl *tally, sim *PoolSimulator, b *fakeBook, in, out string, x *big.Int) *big.Int {
		tl.sizes++
		res, err := calc(sim, in, out, x, nil)
		if err != nil {
			return nil
		}
		want, err := b.quote(x)
		require.NoError(t, err, "the simulator quotes %s where the door refuses", x)
		short := new(big.Int).Sub(want, res.TokenAmountOut.Amount)
		require.GreaterOrEqual(t, short.Sign(), 0, "at %s the simulator pays %s, the door %s", x,
			res.TokenAmountOut.Amount, want)
		ppm, _ := new(big.Rat).SetFrac(new(big.Int).Mul(short, big.NewInt(bookPPM)), want).Float64()
		tl.quoted++
		tl.sumPpm += ppm
		tl.worstPpm = max(tl.worstPpm, ppm)
		if tl.worst == nil || short.Cmp(tl.worst) > 0 {
			tl.worst = short
		}
		return short
	}
	rng := rand.New(rand.NewSource(23))
	var before, after tally
	books := 0
	for trial := range 400 {
		b := randomBook(rng, trial%2 == 0)
		old, edged := refreshed(b, false), refreshed(b, true)
		if len(edged) < 2 {
			continue
		}
		books++
		simOld, in, out := bookSim(t, b, old)
		simNew, _, _ := bookSim(t, b, edged)
		lo, hi := edged[0][0].ToBig(), edged[len(edged)-1][0].ToBig()
		span := new(big.Int).Sub(hi, lo)
		for range 150 {
			x := new(big.Int).Add(lo, new(big.Int).Rand(rng, new(big.Int).Add(span, big.NewInt(1))))
			measure(&before, simOld, b, in, out, x)
			short := measure(&after, simNew, b, in, out, x)
			require.NotNil(t, short, "trial %d: %s is inside the ladder and not quoted", trial, x)
			require.LessOrEqual(t, short.Cmp(big.NewInt(1)), 0,
				"trial %d: at %s the chord is %s units under the door", trial, x, short)
		}
	}
	require.Greater(t, books, 300)
	require.Equal(t, after.sizes, after.quoted, "every size inside the ladder is quoted")
	t.Logf("%d sides, %d sizes each grid, from the first point to the last of the grid with the boundaries", books,
		after.sizes)
	t.Logf("grid as before:       quoted %d, worst shortfall %s units (%.1f ppm of the output), mean %.2f ppm",
		before.quoted, before.worst, before.worstPpm, before.sumPpm/float64(max(before.quoted, 1)))
	t.Logf("with the boundaries:  quoted %d, worst shortfall %s units (%.4f ppm of the output), mean %.6f ppm",
		after.quoted, after.worst, after.worstPpm, after.sumPpm/float64(max(after.quoted, 1)))
	assert.LessOrEqual(t, after.worst.Cmp(big.NewInt(1)), 0)
	assert.Greater(t, before.worst.Cmp(big.NewInt(1)), 0, "the grid without the boundaries gives up more")
}

// rungChain is trackedChain with listing 1's rungs in its words, as the front stores them: selling WETH, four rungs
// of which the fill memory used up the first and part of the second, so the walk changes rung at 0.87, 2.13 and 4.71
// WETH; buying with USDC, two rungs under an older post's memory, at 3,300 and 9,700 USDC.
func rungChain() (*fakeChain, [2][]*big.Int) {
	chain := trackedChain()
	l := chain.listings[1]
	bid := &fakeBook{bid: true, sizeExp: 3, unitExp: 13, sizes: []uint64{40, 137, 263, 521},
		offsets: []uint64{0, 0, 300, 900}, nonce: 5, tag: 5, filled: 50_000}
	ask := &fakeBook{sizeExp: 2, unitExp: 4, sizes: []uint64{3_300, 9_700}, offsets: []uint64{0, 500}, nonce: 6, tag: 4,
		filled: 1_234}
	l.view.PairWord = new(big.Int).Or(big.NewInt(1<<8), new(big.Int).Or(shl(13, 112), shl(4, 120)))
	l.view.Depth[0], l.view.Depth[1] = bid.depth()
	l.view.Depth[2], l.view.Depth[3] = ask.depth()
	l.boards = [2]*big.Int{bid.board(), ask.board()}
	l.sides = [2]fakeSide{
		{rungs: []fakeRung{{87e16, 3, 1e9}, {213e16, 2997, 1e12}, {471e16, 2990, 1e12}}, minOut: 1e6},
		{rungs: []fakeRung{{3_300e6, 333_000_333, 1}, {9_700e6, 332_000_000, 1}}, minOut: 1e14},
	}
	return chain, [2][]*big.Int{
		{big.NewInt(87e16), big.NewInt(213e16), big.NewInt(471e16)},
		{big.NewInt(3_300e6), big.NewInt(9_700e6)},
	}
}

// quotedSizes are the sizes the door was asked to quote in each direction, in the calls from index from on.
func quotedSizes(t *testing.T, n *node, from int) [2]map[string]bool {
	t.Helper()
	sizes := [2]map[string]bool{{}, {}}
	for _, c := range n.calls()[from:] {
		m, err := doorABI.MethodById(c.data)
		if len(c.data) < 4 || err != nil || m.Name != "quote" {
			continue
		}
		args, err := m.Inputs.Unpack(c.data[4:])
		require.NoError(t, err)
		dir := 0
		if args[0] == any(testUSDC) {
			dir = 1
		}
		sizes[dir][args[2].(*big.Int).String()] = true
	}
	return sizes
}

// The tracker asks the door at every rung boundary, read from the words of the head read and no request of its own,
// and keeps the door's answer there: the ladder holds each boundary, and the simulator is the door to a unit between
// them.
func TestTrackerSamplesTheRungEdges(t *testing.T) {
	chain, edges := rungChain()
	n := chain.node(t, 1234)
	p := listed(t, n)
	before := len(n.calls())
	got := track(t, n, p)

	plain := trackedChain()
	plain.listings[1].sides = chain.listings[1].sides
	m := plain.node(t, 1234)
	q := listed(t, m)
	// the same requests as a listing whose words give no boundary: the words ride in the head read's multicall
	plainMultis := len(m.multis)
	track(t, m, q)
	nMultis := len(n.multis)
	track(t, n, p)
	assert.Equal(t, len(m.multis)-plainMultis, len(n.multis)-nMultis, "the boundaries cost no request of their own")

	e := extraOf(t, got)
	asked := quotedSizes(t, n, before)
	for dir := range 2 {
		at := map[string]*uint256.Int{}
		for _, pt := range e.Ladders[dir] {
			at[pt[0].Dec()] = pt[1]
		}
		for _, edge := range edges[dir] {
			assert.True(t, asked[dir][edge.String()], "direction %d: the door is asked at %s", dir, edge)
			want, err := chain.listings[1].sides[dir].quote(edge)
			require.NoError(t, err)
			require.NotNil(t, at[edge.String()], "direction %d: no point at %s", dir, edge)
			assert.Equal(t, want, at[edge.String()].ToBig(), "direction %d at %s", dir, edge)
		}
		last := e.Ladders[dir][len(e.Ladders[dir])-1][0].ToBig()
		assert.Equal(t, edges[dir][len(edges[dir])-1], last, "direction %d: the ladder ends where the rungs do", dir)
	}

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	tokens := [2]string{hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])}
	rng := rand.New(rand.NewSource(5))
	for dir := range 2 {
		first := e.Ladders[dir][0][0].ToBig()
		span := new(big.Int).Sub(edges[dir][len(edges[dir])-1], first)
		for range 1500 {
			x := new(big.Int).Add(first, new(big.Int).Rand(rng, span))
			res, err := calc(sim, tokens[dir], tokens[1-dir], x, nil)
			require.NoError(t, err, "direction %d at %s", dir, x)
			want, err := chain.listings[1].sides[dir].quote(x)
			require.NoError(t, err)
			short := new(big.Int).Sub(want, res.TokenAmountOut.Amount)
			require.GreaterOrEqual(t, short.Sign(), 0, "direction %d at %s: above the door", dir, x)
			require.LessOrEqual(t, short.Cmp(big.NewInt(1)), 0, "direction %d at %s: %s under the door", dir, x, short)
		}
	}
}

// The words are read by their layout only under an implementation the front's config lists. Under any other the
// tracker asks the door at exactly the sizes it asked before this reading existed, and quotes nothing as before; and a
// front that does not answer the storage read is sampled on that same grid and still quoted.
func TestTrackerKeepsTheOldGridUnderAnUnknownImplementation(t *testing.T) {
	run := func(set func(*fakeChain)) (asked [2]map[string]bool, e Extra) {
		chain, _ := rungChain()
		set(chain)
		n := chain.node(t, 1234)
		p := listed(t, n)
		before := len(n.calls())
		got := track(t, n, p)
		return quotedSizes(t, n, before), extraOf(t, got)
	}
	_, edges := rungChain()
	listedImpl, withEdges := run(func(*fakeChain) {})
	unknown, none := run(func(c *fakeChain) { c.impl = common.HexToAddress("0x6000000000000000000000000000000000000002") })
	silent, old := run(func(c *fakeChain) { c.noSloads = true })

	for dir := range 2 {
		assert.NotEmpty(t, silent[dir])
		assert.Equal(t, silent[dir], unknown[dir], "direction %d: an unknown implementation, the grid as it was", dir)
		assert.Empty(t, none.Ladders[dir], "direction %d: and no quote under it", dir)
		assert.NotEmpty(t, old.Ladders[dir], "direction %d: a front with no storage read is still quoted", dir)
		want := map[string]bool{}
		for s := range silent[dir] {
			want[s] = true
		}
		prev := new(big.Int)
		for k, edge := range edges[dir] {
			// the last boundary is the side's top, which the grid held already
			assert.Equal(t, k == len(edges[dir])-1, silent[dir][edge.String()],
				"direction %d: of the boundaries, the grid as it was holds the top alone (%s)", dir, edge)
			mid := new(big.Int).Add(prev, edge)
			want[edge.String()], want[mid.Rsh(mid, 1).String()] = true, true
			prev = edge
		}
		assert.Equal(t, want, listedImpl[dir],
			"direction %d: the listed implementation adds the boundaries and each rung's middle alone", dir)
		assert.Greater(t, len(withEdges.Ladders[dir]), len(old.Ladders[dir]))
	}
}

// The minimum fill is read from the listing's settings word as the front packs it, and the sizes at it are the input
// itself with the quote in, and with the base in the two inputs that bracket the least one paying it, from the last
// ladder's first point.
func TestMinSizes(t *testing.T) {
	assert.Zero(t, minFill(nil).Sign())
	assert.Equal(t, big.NewInt(50e6), minFill(shl(5|7<<18, 160)), "mantissa 5, exponent 7")
	all := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	all.Xor(all, shl(1<<24-1, 160))
	assert.Zero(t, minFill(all).Sign(), "no other bit of the word is read")
	field, amount := amount24(big.NewInt(123_456_789))
	assert.Equal(t, big.NewInt(123_456_000), amount)
	assert.Equal(t, amount, minFill(shl(field, 160)))

	v := zeroView()
	v.PairWord = big.NewInt(1 << 8) // the base is token 0: direction 0 has the base in
	r := rungRead{settings: shl(5|7<<18, 160), ok: true}
	assert.Equal(t, []*big.Int{big.NewInt(50e6)}, minSizes(v, 1, r, Point{}), "the quote in: the input itself")
	assert.Empty(t, minSizes(v, 0, r, Point{}), "the base in, no ladder yet")
	// 0.03 WETH paid 90.000001 USDC: 50 USDC takes over 50e6 * 3e16 / 90000002 and at most 50e6 * 3e16 / 90000001
	first := Point{uint256.NewInt(3e16), uint256.NewInt(90_000_001)}
	assert.Equal(t, []*big.Int{bi("16666666296296305"), bi("16666666481481484")}, minSizes(v, 0, r, first))
	v.PairWord = new(big.Int) // the base is token 1
	assert.Equal(t, []*big.Int{big.NewInt(50e6)}, minSizes(v, 0, r, Point{}))
	assert.Empty(t, minSizes(v, 0, rungRead{settings: new(big.Int), ok: true}, first), "no minimum")
	assert.Empty(t, minSizes(v, 0, rungRead{settings: r.settings}, first), "no storage read")
}

// With the sizes at the minimum fill the ladder starts where the listing does: with the quote in at the minimum
// itself from the first refresh, with the base in within a few refreshes at an input the door refuses a little
// under. Without them it starts at the grid's first size the door fills, and every smaller fill goes unquoted.
func TestLadderStartsAtTheMinimumFill(t *testing.T) {
	rng := rand.New(rand.NewSource(41))
	var asks, bids int
	var before, after float64
	for trial := range 600 {
		b := randomBook(rng, trial%2 == 0)
		b.outBound = nil
		if b.minFill == nil || b.minFill.Sign() == 0 {
			continue
		}
		old := refreshed(b, false)
		points := bookLadder(b, bookGrid(b, nil, true))
		if len(old) == 0 || len(points) == 0 {
			continue
		}
		least := b.minFill
		if b.bid {
			for range 3 {
				points = bookLadder(b, bookGrid(b, points, true))
			}
			// the least input that pays the minimum: the door fills the first point and refuses 0.1% under it
			x := points[0][0].ToBig()
			under := new(big.Int).Sub(x, new(big.Int).Add(new(big.Int).Quo(x, big.NewInt(1000)), big.NewInt(2)))
			if _, err := b.quote(under); under.Sign() > 0 {
				require.ErrorIs(t, err, errBelowMinimum, "trial %d: the ladder starts at %s and the door fills %s", trial,
					x, under)
			}
			least = x
			bids++
		} else {
			if _, err := b.quote(b.minFill); err != nil {
				continue // the rungs pay nothing for the minimum
			}
			require.Equal(t, b.minFill, points[0][0].ToBig(), "trial %d: the ladder starts at the minimum", trial)
			asks++
		}
		was, _ := new(big.Rat).SetFrac(old[0][0].ToBig(), least).Float64()
		now, _ := new(big.Rat).SetFrac(points[0][0].ToBig(), least).Float64()
		before, after = before+was, after+now
	}
	assert.Greater(t, asks, 150)
	assert.Greater(t, bids, 150)
	t.Logf("%d sides: the first point is %.2f times the least fill in the mean on the grid as before, %.4f with the "+
		"sizes at the minimum", asks+bids, before/float64(asks+bids), after/float64(asks+bids))
}

// The tracker reads the minimum fill in the head read's storage call and starts each ladder there: 50 USDC in with
// the quote in, and with the base in, from the second refresh, the least WETH that pays 50 USDC.
func TestTrackerStartsTheLadderAtTheMinimumFill(t *testing.T) {
	chain, _ := rungChain()
	l := chain.listings[1]
	l.settings = shl(5|7<<18, 160)
	l.sides[0].minOut, l.minIn[1] = 50e6, 50e6
	n := chain.node(t, 1234)
	p := track(t, n, listed(t, n))
	e := extraOf(t, p)
	assert.Equal(t, "50000000", e.Ladders[1][0][0].Dec(), "the quote in starts at the minimum")
	assert.Greater(t, e.Ladders[0][0][0].Uint64(), uint64(17e15), "the base in starts on the grid at first")

	multis := len(n.multis)
	e = extraOf(t, track(t, n, p))
	first := e.Ladders[0][0]
	assert.GreaterOrEqual(t, first[1].Uint64(), uint64(50e6))
	_, err := l.sides[0].quote(new(big.Int).Sub(first[0].ToBig(), big.NewInt(1e9)))
	assert.ErrorIs(t, err, errBelowMinimum, "a billionth of a WETH less pays under the minimum")

	// under an implementation not listed the settings are not read: the grid as it was
	other, _ := rungChain()
	other.listings[1].settings, other.listings[1].minIn[1] = l.settings, 50e6
	other.impl = common.HexToAddress("0x6000000000000000000000000000000000000002")
	m := other.node(t, 1234)
	q := listed(t, m)
	from := len(m.calls())
	track(t, m, q)
	assert.False(t, quotedSizes(t, m, from)[1]["50000000"])
	// and the read is no request of its own
	plain := trackedChain()
	plain.listings[1].sides = l.sides
	k := plain.node(t, 1234)
	r := track(t, k, listed(t, k))
	kMultis := len(k.multis)
	track(t, k, r)
	assert.Equal(t, len(k.multis)-kMultis, len(n.multis)-multis)
}

// rungEthChain is rungChain's listing 1 under the Ethereum config, with a credit mode where credit is set (as
// creditChain's: its WETH-in side routed past 2 WETH).
func rungEthChain(credit bool) *fakeChain {
	rung, _ := rungChain()
	chain := creditChain()
	if !credit {
		chain = ethChain()
	}
	l, r := chain.listings[1], rung.listings[1]
	l.view.PairWord, l.view.Depth, l.boards, l.sides = r.view.PairWord, r.view.Depth, r.boards, r.sides
	return chain
}

// askedSizes is the sizes a refresh of rungEthChain's listing asks the door at, per direction, with the config's front
// running the implementation given (unlisted: the grid as it was).
func askedSizes(t *testing.T, credit bool, impl common.Address) [2]map[string]bool {
	t.Helper()
	chain := rungEthChain(credit)
	chain.impl = impl
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	if credit {
		setCredit(cfg, testCredit)
	}
	p := ethListed(t, n, cfg)
	before := len(n.calls())
	ethTrack(t, n, cfg, p)
	return quotedSizes(t, n, before)
}

// A listing with a credit mode keeps the grid it had: it takes one fill a route whatever the state says, so the rung
// middles and the minimum fill's sizes buy it nothing, and each size there costs a plan beside its quote in the one
// pinned call. A listing without one takes the new grid, which is larger.
func TestCreditListingKeepsTheOldGrid(t *testing.T) {
	unlisted := common.HexToAddress("0x6000000000000000000000000000000000000002")
	oldGrid := askedSizes(t, false, unlisted)
	custody := askedSizes(t, false, common.Address{})
	routed := askedSizes(t, true, common.Address{})
	routedOld := askedSizes(t, true, unlisted)
	for dir := range 2 {
		assert.LessOrEqual(t, len(oldGrid[dir]), 22, "direction %d: the old grid's most", dir)
		assert.Greater(t, len(custody[dir]), len(oldGrid[dir]), "direction %d: a custody listing takes the new grid", dir)
		assert.LessOrEqual(t, len(routed[dir]), 22, "direction %d: a credit listing stays at the old grid's most", dir)
		assert.Equal(t, routedOld[dir], routed[dir], "direction %d: a credit listing asks the sizes it asked before", dir)
	}
}
