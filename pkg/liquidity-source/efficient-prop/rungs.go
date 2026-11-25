package efficientprop

import (
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// The front fills a side by walking its maker's rungs from the side's fill memory, at most ten, each at one price
// (PropAMMFront._walk), and floors the output once. So between two rung boundaries the output is one straight line of
// the input, and a ladder with a point on every boundary has a floored chord that is the fill to a unit. This file
// finds the boundaries, as sizes for the door's own quote: it prices nothing.
//
// A side's rungs are in its two depth words, which doorView answers: word 0 holds the size exponent (bits 112-117),
// whether word 1 is present (bit 118) and rungs 0 to 2 (36 bits each from bit 120), word 1 rungs 3 to 9 (from bit 0).
// A rung is its cumulative size (16 bits, low) and its offset. A size counts 10^exponent board units, and a board
// unit is 10^e base units of the side's input token, e the pair word's board exponent of the side (bits 112-119 for
// the base in, 120-127 for the quote in). The fill memory is the side's board word, record word 8 for the base in and
// 9 for the quote in: its depth tag (bits 4-51) and the board units filled (bits 52-91), which count only while the
// tag is the depth word's nonce (bits 12-59), and not where the standing word refills the side (refills).
const (
	boardBidWord = 8
	boardAskWord = 9

	maxRungs = 10
	// the most the front writes: a depth post's size exponent, and a pair's board exponents
	maxDepthExp = 30
	maxBoardExp = 16

	// ctlRefill is the controls word's flag that lets a standing word refill a side (CTL_REFILL)
	ctlRefill = 66
)

// slotsABI is the front's raw storage read, extSloads: the words at the slots asked, in order.
var slotsABI = mustABI([]byte(`[{"type":"function","name":"extSloads","stateMutability":"view",` +
	`"inputs":[{"name":"slots","type":"bytes32[]"}],"outputs":[{"name":"","type":"bytes32[]"}]}]`))

// rungRead is what the rung boundaries need beside doorView, read from the front's storage in the head read: the two
// sides' board words (0: the base in, 1: the quote in), the listing's controls word and the front proxy's
// implementation, all at the head read's block. ok is whether the front answered all four.
type rungRead struct {
	board          [2]*big.Int
	controls       *big.Int
	implementation common.Address
	ok             bool
}

// rungSlots are the slots of a listing's rungRead, in its order.
func rungSlots(listing uint32) [][32]byte {
	return [][32]byte{listingSlot(listing, boardBidWord), listingSlot(listing, boardAskWord),
		controlsSlot(listing), implementationSlot}
}

// rungCall is the head read's call for a listing's rungRead: one more call of the head multicall, no request of its
// own.
func rungCall(front common.Address, listing uint32) call {
	return call{abi: &slotsABI, target: front, method: "extSloads", params: []any{rungSlots(listing)}}
}

// rungReadOf is a rungRead from extSloads' answer for rungSlots; not ok for any other answer.
func rungReadOf(words [][32]byte) (r rungRead) {
	if len(words) != 4 {
		return r
	}
	for i := range r.board {
		r.board[i] = new(big.Int).SetBytes(words[i][:])
	}
	r.controls = new(big.Int).SetBytes(words[2][:])
	r.implementation = common.BytesToAddress(words[3][:])
	r.ok = true
	return r
}

// unpackRungRead is rungCall's return data as a rungRead; not ok where the call reverted or answered something else.
func unpackRungRead(ret []byte) rungRead {
	if ret == nil {
		return rungRead{}
	}
	vals, err := slotsABI.Methods["extSloads"].Outputs.Unpack(ret)
	if err != nil || len(vals) != 1 {
		return rungRead{}
	}
	words, _ := vals[0].([][32]byte)
	return rungReadOf(words)
}

// sideIsBid is whether direction dir (0: token0 in) sells the pair's base, the side the front calls the bid: the pair
// word's bit 8 says the base is token 0.
func sideIsBid(pairWord *big.Int, dir int) bool {
	return (dir == 0) == (pairWord != nil && pairWord.Bit(8) == 1)
}

// boardExp is the board exponent of a side's input token, from the pair word; not ok above the most the front writes.
func boardExp(pairWord *big.Int, bid bool) (uint8, bool) {
	lo := uint(120)
	if bid {
		lo = 112
	}
	e := bits(pairWord, lo, 8)
	return uint8(e), pairWord != nil && e <= maxBoardExp
}

// refills is whether the standing word refills a side whose board word is board, as the front judges it
// (PropAMMWords.refills, under the controls' flag): the word asks it (bit 232 for the base in, 233 for the quote in),
// acknowledges the side's fill sequence (11 bits from 234 or 245 against the board's bits 236-246), was signed after
// the side's last fill (its maker time, bits 72-119, over the board's time, bits 188-235) and is not the word that
// refilled the side last (its time's low 9 bits against the board's bits 247-255). A refilled side fills from the
// start of its rungs.
func refills(word, board, controls *big.Int, bid bool) bool {
	if word == nil || controls == nil || controls.Bit(ctlRefill) == 0 {
		return false
	}
	on, ack := word.Bit(233) == 1, bits(word, 245, 11)
	if bid {
		on, ack = word.Bit(232) == 1, bits(word, 234, 11)
	}
	ts := bits(word, 72, 48)
	return on && ack == bits(board, 236, 11) && ts > bits(board, 188, 48) && ts&0x1ff != bits(board, 247, 9)
}

// rungEdges are direction dir's rung boundaries at the head read: for each rung the side's fill memory has not used
// up, the cumulative input at which the front's walk leaves it, ascending. None where the side's words are not the
// layout this reads (a depth word of another version, an exponent above the front's bounds), which leaves the grid as
// it was. They are sizes to ask the door at, never prices: a boundary read wrong moves a sampled size and no output.
func rungEdges(v DoorView, dir int, r rungRead) []*big.Int {
	if !r.ok || v.PairWord == nil {
		return nil
	}
	bid := sideIsBid(v.PairWord, dir)
	d0, d1, board := v.Depth[2], v.Depth[3], r.board[1]
	if bid {
		d0, d1, board = v.Depth[0], v.Depth[1], r.board[0]
	}
	unitExp, ok := boardExp(v.PairWord, bid)
	sizeExp := bits(d0, 112, 6)
	if d0 == nil || !ok || bits(d0, 0, 4) != 1 || sizeExp > maxDepthExp {
		return nil
	}
	ten := big.NewInt(10)
	unit := new(big.Int).Exp(ten, big.NewInt(int64(unitExp)), nil)
	scale := new(big.Int).Exp(ten, new(big.Int).SetUint64(sizeExp), nil)
	scale.Mul(scale, unit)
	// the fill memory counts only against the depth post it was filled from, and not past a refill
	filled := uint64(0)
	if bits(board, 4, 48) == bits(d0, 12, 48) && !refills(v.Word, board, r.controls, bid) {
		filled = bits(board, 52, 40)
	}
	cursor := new(big.Int).Mul(new(big.Int).SetUint64(filled), unit)

	var edges []*big.Int
	for j := range uint(maxRungs) {
		if j == 3 && (d0.Bit(118) == 0 || d1 == nil) {
			break
		}
		rung := bits(d0, 120+36*j, 36)
		if j >= 3 {
			rung = bits(d1, 36*(j-3), 36)
		}
		size := rung & 0xffff
		if size == 0 {
			break
		}
		h := new(big.Int).Mul(new(big.Int).SetUint64(size), scale)
		if h.Cmp(cursor) <= 0 {
			continue
		}
		h.Sub(h, cursor)
		// sizes rise strictly in a post the front took: one that does not is no boundary
		if len(edges) > 0 && h.Cmp(edges[len(edges)-1]) <= 0 {
			continue
		}
		edges = append(edges, h)
	}
	return edges
}

// withEdges is sizes (ascending, distinct) with each rung boundary the front takes as an input in its place, and the
// middle of each rung: a rung then holds two spans of its one price, so a fill that starts inside it has a span of
// that price on one side, which is what prices the fill after it to a few units (curve.quoteAfter).
func withEdges(sizes, edges []*big.Int) []*big.Int {
	prev := new(big.Int)
	for _, e := range edges {
		if e.Sign() <= 0 || e.Cmp(maxPointInput) >= 0 {
			continue
		}
		sizes = withSize(sizes, e)
		if mid := new(big.Int).Add(prev, e); mid.Rsh(mid, 1).Cmp(prev) > 0 {
			sizes = withSize(sizes, mid)
		}
		prev = e
	}
	return sizes
}

// repeatOf is a listing's Repeat: for each direction with a ladder, its input's board unit and what its ladder proves
// of a later fill in one route (repeatKind). Nil where no direction's proves anything.
func repeatOf(v DoorView, sizes [2][]*big.Int, ladders [2][]Point, samples [2][][]*big.Int) *Repeat {
	var r Repeat
	for dir := range 2 {
		// the side's unit, and its depth post of the version whose fill memory this reads (rungEdges)
		bid := sideIsBid(v.PairWord, dir)
		d0 := v.Depth[2]
		if bid {
			d0 = v.Depth[0]
		}
		exp, ok := boardExp(v.PairWord, bid)
		if !ok || bits(d0, 0, 4) != 1 {
			continue
		}
		r.Unit[dir], r.Kind[dir] = exp, repeatKind(sizes[dir], ladders[dir], samples[dir])
	}
	if r.Kind == [2]uint8{} {
		return nil
	}
	return &r
}

// repeatKind is what a direction's ladder proves of a later fill, from the samples it was kept from (each the door's
// answers at sizes, nil where it refused; the ladder a run of sizes with the least of them).
//
// Where every sample answered the ladder itself at each of its points, the ladder is each block's own, the floors
// of that block's one concave walk (repeatOwn): one block sampled, or several over which no level moved. Else the
// ladder is steady (repeatSteady) where in every sample the excess over the ladder never falls by more than a unit as
// the size grows, so between any two points that block pays at least the ladder's rise less a unit, which is what the
// simulator's bound from the right needs of each block (curve.quoteAfter). The front adds one discount to every
// rung's offset, so the block that pays least pays least on every rung and the excess of any other only grows.
// Anything else is 0: the direction takes one fill a route.
func repeatKind(sizes []*big.Int, run []Point, samples [][]*big.Int) uint8 {
	if len(run) == 0 || len(samples) == 0 {
		return 0
	}
	first := run[0][0].ToBig()
	start := slices.IndexFunc(sizes, func(s *big.Int) bool { return s.Cmp(first) == 0 })
	if start < 0 || start+len(run) > len(sizes) {
		return 0
	}
	own := true
	for _, sample := range samples {
		if len(sample) != len(sizes) {
			return 0
		}
		var most *big.Int
		for k, pt := range run {
			out := sample[start+k]
			if sizes[start+k].Cmp(pt[0].ToBig()) != 0 || out == nil {
				return 0
			}
			excess := new(big.Int).Sub(out, pt[1].ToBig())
			if excess.Sign() < 0 {
				return 0
			}
			own = own && excess.Sign() == 0
			if most == nil || excess.Cmp(most) > 0 {
				most = excess
			} else if new(big.Int).Sub(most, excess).Cmp(big.NewInt(1)) > 0 {
				return 0
			}
		}
	}
	if own {
		return repeatOwn
	}
	return repeatSteady
}
