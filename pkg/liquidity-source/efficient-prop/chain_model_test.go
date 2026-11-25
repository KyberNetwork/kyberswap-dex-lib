package efficientprop

// A model of the deployed contracts for the property test and the fixed cases (chain_prop_test.go, chain_cases_test.go).
//
// An independent model of the deployed contracts, written from the Solidity and from nothing in this package:
//
//	PropAMMFront.sol   _evaluate :1305-1533, _walk :1694-1752, _firstLevel :1657-1678, _unclamped :1682-1688,
//	                   _fence :1644-1654, _quoteTail :512-530, _quoteFillable :586-610, _swapNamed :702-748
//	PropAMMShared.sol  _bucket :132-148, _band :385-396
//	PropAMMWords.sol   pool word, depth, controls, board, pair word, settings (minFill :552, splitDepth :556)
//	PropAMMLaneLib.sol amount24 :42, spreadFee :20-39
//	FLAMMRoute.sol     fillFromFront :59-102, laneStateOf :118-135, _open :185-210, _fee :239-257
//	FLAMMLane.sol      boundFee :190-206, book :571-587, _debitAt :1042-1052
//	FLAMMStore.sol     reserved :1769, reservedCap :1776, notional :1761, feeBound :1676
//
// Every word is packed and read here with math/big alone.

import (
	"math/big"
)

var (
	chOne      = big.NewInt(1)
	chPPM      = big.NewInt(1_000_000)
	chE36      = new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	chAskScale = new(big.Int).Lsh(big.NewInt(1), 96)
	chAskNum   = new(big.Int).Mul(chE36, chAskScale)
	chCap128   = new(big.Int).Lsh(big.NewInt(1), 128)
	chMax256   = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	chNotional = big.NewInt(10_000_000) // FLAMMStore.NOTIONAL_SCALE
	chWad      = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	chMinBand  = new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil) // MIN_FEE_BAND_WAD 0.01e18
)

func chF(w *big.Int, lo, n uint) *big.Int {
	v := new(big.Int).Rsh(w, lo)
	mask := new(big.Int).Sub(new(big.Int).Lsh(chOne, n), chOne)
	return v.And(v, mask)
}
func chU(w *big.Int, lo, n uint) uint64 { return chF(w, lo, n).Uint64() }
func chPow10(e uint64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), new(big.Int).SetUint64(e), nil)
}
func chN(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
func chMinB(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}
func chShl(v *big.Int, n uint) *big.Int { return new(big.Int).Lsh(v, n) }

// ---------------------------------------------------------------- the pool (a lane pair past the hook pair)

// chPool is the pool as FLAMMRoute reads and writes it for one pair: index 0 the pair's base, 1 its quote.
type chPool struct {
	// ideal: a pool that bounds nothing (room 2^128, no cap, every fill taken)
	ideal    bool
	held     [2]*big.Int // tracked custody
	owed     [2]*big.Int // spread fee owed in the token
	level    [2]*big.Int // outflow bucket level, units
	unit     [2]*big.Int // bucket unit, base units
	capN     [2]*big.Int // notional cap (0 none)
	rate     uint64      // the pair's notional rate, parts of 1e7 (0 at launch)
	spreadBp uint64      // the pool's spread rate, bp (0 at launch)
	bandWad  *big.Int
}

func chReserved(cash *big.Int, rate uint64) *big.Int {
	if rate == 0 {
		return new(big.Int).Set(cash)
	}
	v := new(big.Int).Mul(cash, chNotional)
	return v.Quo(v, new(big.Int).Add(chNotional, chN(rate)))
}

func chReservedCap(c *big.Int, rate uint64) *big.Int {
	r := chReserved(c, rate)
	if r.Sign() == 0 {
		return new(big.Int).Set(c)
	}
	return r
}

// cash is _open's: the custody of y less the fee owed in it, reserved where the part is owed in y.
func (p *chPool) cash(y int, owedInY bool) *big.Int {
	c := new(big.Int).Sub(p.held[y], p.owed[y])
	if c.Sign() < 0 {
		c.SetInt64(0)
	}
	if p.rate != 0 && owedInY {
		c = chReserved(c, p.rate)
	}
	return c
}

// laneStateOf is FLAMMRoute.laneStateOf :118-135.
func (p *chPool) laneStateOf(bid bool) (room, maxIn *big.Int) {
	if p.ideal {
		return new(big.Int).Set(chCap128), new(big.Int)
	}
	y := 0
	if bid {
		y = 1
	}
	x := 1 - y
	cash := p.cash(y, bid)
	maxIn = chReservedCap(p.capN[x], p.rate)
	room = chReserved(new(big.Int).Mul(p.level[y], p.unit[y]), p.rate)
	if !bid { // q != y
		cash = chReserved(cash, p.rate)
	}
	if cash.Cmp(room) < 0 {
		room = cash
	}
	if c := chReservedCap(p.capN[y], p.rate); c.Sign() != 0 && c.Cmp(room) < 0 {
		room = c
	}
	return room, maxIn
}

// fill is FLAMMRoute.fillFromFront :59-102; "" where the pool takes the fill, else its revert.
func (p *chPool) fill(bid bool, in, out, p36 *big.Int) string {
	y := 0
	if bid {
		y = 1
	}
	x := 1 - y
	if p.ideal {
		p.held[x] = new(big.Int).Add(p.held[x], in)
		p.held[y] = new(big.Int).Sub(p.held[y], out)
		return ""
	}
	if out.Cmp(p.cash(y, bid)) > 0 {
		return "OutputAboveCeiling"
	}
	if c := chReservedCap(p.capN[x], p.rate); c.Sign() != 0 && in.Cmp(c) > 0 {
		return "NotionalCap(in)"
	}
	if c := chReservedCap(p.capN[y], p.rate); c.Sign() != 0 && out.Cmp(c) > 0 {
		return "NotionalCap(out)"
	}
	hx, hy := new(big.Int).Add(p.held[x], in), new(big.Int).Sub(p.held[y], out)
	hq := hx
	if bid {
		hq = hy
	}
	fee := p.fee(bid, in, out, p36, hq)
	// FLAMMLane.book: the payout debited from y's bucket, rounded up to its unit
	debit, rem := new(big.Int).QuoRem(out, p.unit[y], new(big.Int))
	if rem.Sign() != 0 {
		debit.Add(debit, chOne)
	}
	if debit.Cmp(p.level[y]) > 0 {
		return "OutflowCap"
	}
	p.level[y] = new(big.Int).Sub(p.level[y], debit)
	p.owed[1] = new(big.Int).Add(p.owed[1], fee)
	p.held[x], p.held[y] = hx, hy
	return ""
}

// fee is FLAMMRoute._fee :239-257 with PropAMMLaneLib.spreadFee and FLAMMLane.boundFee.
func (p *chPool) fee(bid bool, in, out, p36, hq *big.Int) *big.Int {
	fee := new(big.Int)
	ratePpm := p.spreadBp * 100
	if p36.Sign() != 0 && ratePpm != 0 {
		if bid {
			value := new(big.Int).Mul(in, p36)
			value.Quo(value, chE36)
			if value.Cmp(out) > 0 {
				fee.Sub(value, out).Mul(fee, chN(ratePpm)).Quo(fee, chPPM)
			}
		} else {
			cost := new(big.Int).Mul(out, p36)
			cost.Add(cost, new(big.Int).Sub(chE36, chOne)).Quo(cost, chE36)
			if in.Cmp(cost) > 0 {
				fee.Sub(in, cost).Mul(fee, chN(ratePpm)).Quo(fee, chPPM)
			}
		}
	}
	if fee.Sign() == 0 && p.rate == 0 {
		return fee
	}
	room := new(big.Int).Sub(hq, p.owed[1])
	if room.Sign() < 0 {
		room.SetInt64(0)
	}
	leg := in
	if bid {
		leg = out
	}
	if fee.Sign() != 0 {
		band := p.bandWad
		if band.Cmp(chMinBand) < 0 {
			band = chMinBand
		}
		most := new(big.Int).Mul(leg, band)
		most.Mul(most, chN(p.spreadBp)).Quo(most, new(big.Int).Mul(chWad, big.NewInt(10_000)))
		fee = chMinB(fee, most)
		if bid {
			most = new(big.Int).Mul(leg, chN(p.spreadBp))
			most.Quo(most, chN(10_000-p.spreadBp))
			fee = chMinB(fee, most)
		}
	}
	n := new(big.Int).Mul(leg, chN(p.rate))
	n.Quo(n, chNotional)
	if n.Cmp(fee) > 0 {
		fee = n
	}
	return new(big.Int).Set(chMinB(fee, room))
}

func (p *chPool) clone() *chPool {
	c := *p
	for i := range 2 {
		c.held[i], c.owed[i], c.level[i] = new(big.Int).Set(p.held[i]), new(big.Int).Set(p.owed[i]),
			new(big.Int).Set(p.level[i])
	}
	return &c
}

// ---------------------------------------------------------------- one side of a listing at the front

// chSide is one side of a listing as the front stores it. The words are the contract's own packing.
type chSide struct {
	bid bool
	// depth words 0 and 1, the side's board word
	w0, w1, board *big.Int
	// shared by the listing's two sides
	word, ctl, pair, s1 *big.Int
	// the anchor a (the word's mantissa times the pair's price unit) and the print P (0: the listing reads none),
	// price36; conf the print's confidence, ppm
	a, P *big.Int
	conf uint64
	// the desk bucket's rate (quote base units a second) and burst, as _bucket has them after its mins
	bRate, bBurst *big.Int
	pool          *chPool
}

// chBlock is what a block fixes for a fill: the front's discount c (PropAMMFront._discount) and its clock.
type chBlock struct {
	c     uint64
	nowMs uint64
}

const (
	chQuote    = 0
	chFillable = 1
	chSwap     = 2
)

type chPlan struct {
	err    string
	x, out *big.Int
	meets  bool
	board  *big.Int
}

func chAmount24(v uint64) *big.Int {
	m := new(big.Int).SetUint64(v & 0x3ffff)
	return m.Mul(m, chPow10((v>>18)&0x3f))
}

func (s *chSide) minFill() *big.Int { return chAmount24(chU(s.s1, 160, 24)) }

// refills is PropAMMWords.refills :78-82.
func (s *chSide) refills(b *big.Int) bool {
	on, ack := s.word.Bit(233) == 1, chU(s.word, 245, 11)
	if s.bid {
		on, ack = s.word.Bit(232) == 1, chU(s.word, 234, 11)
	}
	ts := chU(s.word, 72, 48)
	return on && ack == chU(b, 236, 11) && ts > chU(b, 188, 48) && ts&0x1ff != chU(b, 247, 9)
}

func (s *chSide) rung(j uint) uint64 {
	if j < 3 {
		return chU(s.w0, 120+36*j, 36)
	}
	return chU(s.w1, 36*(j-3), 36)
}

// unclamped is PropAMMFront._unclamped :1682-1688.
func (s *chSide) unclamped(d uint64) *big.Int {
	if s.bid {
		v := new(big.Int).Mul(s.a, chN(1_000_000-d))
		return v.Quo(v, chPPM)
	}
	den := chN(1_000_000 - d)
	v := new(big.Int).Mul(s.a, chPPM)
	v.Add(v, den).Sub(v, chOne)
	return v.Quo(v, den)
}

// firstLevel is PropAMMFront._firstLevel :1657-1678.
func (s *chSide) firstLevel(scale, cursor *big.Int, c uint64) (bool, *big.Int) {
	for j := uint(0); j < 10; j++ {
		if j == 3 && s.w0.Bit(118) == 0 {
			return false, nil
		}
		rg := s.rung(j)
		size := rg & 0xffff
		if size == 0 {
			return false, nil
		}
		if new(big.Int).Mul(chN(size), scale).Cmp(cursor) <= 0 {
			continue
		}
		d := rg>>16 + c
		if d >= 1_000_000 {
			return false, nil
		}
		return true, s.unclamped(d)
	}
	return false, nil
}

// walk is PropAMMFront._walk :1694-1752.
func (s *chSide) walk(inBound, outBound, scale, cursor, lim *big.Int, c uint64, rungs uint) (x, n *big.Int) {
	x, n = new(big.Int), new(big.Int)
	I := inBound
	if I.Sign() == 0 {
		return x, n
	}
	D := chAskScale
	if s.bid {
		D = chE36
	}
	L := new(big.Int).Add(outBound, chOne)
	L.Mul(L, D).Sub(L, chOne)
	prev := new(big.Int)
	for j := uint(0); j < rungs; j++ {
		if j == 3 && s.w0.Bit(118) == 0 {
			break
		}
		rg := s.rung(j)
		size := rg & 0xffff
		if size == 0 {
			break
		}
		h := new(big.Int).Mul(chN(size), scale)
		if h.Cmp(cursor) <= 0 {
			prev = h
			continue
		}
		d := rg>>16 + c
		if d >= 1_000_000 {
			break
		}
		var q *big.Int
		pu := s.unclamped(d)
		if s.bid {
			q = pu
			if lim.Cmp(pu) <= 0 {
				q = lim
			}
		} else {
			px := pu
			if lim.Cmp(pu) > 0 {
				px = lim
			}
			if px.Sign() == 0 {
				break
			}
			q = new(big.Int).Quo(chAskNum, px)
		}
		if q.Sign() == 0 {
			break
		}
		from := prev
		if cursor.Cmp(prev) > 0 {
			from = cursor
		}
		t := new(big.Int).Sub(h, from)
		if left := new(big.Int).Sub(I, x); t.Cmp(left) > 0 {
			t = left
		}
		roomN := new(big.Int).Sub(L, n)
		if most := new(big.Int).Quo(roomN, q); t.Cmp(most) > 0 {
			t = most
			x.Add(x, t)
			n.Add(n, new(big.Int).Mul(t, q))
			break
		}
		n.Add(n, new(big.Int).Mul(t, q))
		x.Add(x, t)
		if x.Cmp(I) == 0 {
			break
		}
		prev = h
	}
	return x, n
}

// band is PropAMMShared._band :385-396.
func chBand(B, R, k, c uint64) (f, r uint64, wide bool) {
	if c >= R {
		return 0, 0, true
	}
	f = c * k / 10
	if f < B {
		f = B
	}
	if f > R {
		f = R
	}
	r = 2 * f
	if r > R {
		r = R
	}
	return f, r, c > r
}

// evaluate is PropAMMFront._evaluate :1379-1531 for a listing with no credit mode on a lane pair.
func (s *chSide) evaluate(mode int, amountIn *big.Int, blk chBlock) chPlan {
	if chU(s.w0, 0, 4) != 1 {
		return chPlan{err: "DepthExpired"}
	}
	bexp := chU(s.pair, 120, 8)
	if s.bid {
		bexp = chU(s.pair, 112, 8)
	}
	unit := chPow10(bexp)
	scale := new(big.Int).Mul(chPow10(chU(s.w0, 112, 6)), unit)
	b := s.board
	nonce := chF(s.w0, 12, 48)
	filled := new(big.Int)
	if chF(b, 4, 48).Cmp(nonce) == 0 {
		filled = chF(b, 52, 40)
	}
	refilled := s.ctl.Bit(66) == 1 && s.refills(b)
	if refilled {
		filled = new(big.Int)
	}
	cursor := new(big.Int).Mul(filled, unit)
	lim := new(big.Int)
	if s.P.Sign() != 0 {
		B, R, k := chU(s.s1, 16, 16)*10, chU(s.s1, 32, 16)*10, chU(s.s1, 232, 8)
		if s.bid {
			B = chU(s.s1, 0, 16) * 10
		}
		f, rp, wide := chBand(B, R, k, s.conf)
		if wide {
			return chPlan{err: "WidePull"}
		}
		// _fence :1644-1654
		if s.bid {
			lim.Mul(s.P, chN(1_000_000+f)).Quo(lim, chPPM)
		} else {
			if f > 1_000_000 {
				f = 1_000_000
			}
			lim.Mul(s.P, chN(1_000_000-f)).Add(lim, new(big.Int).Sub(chPPM, chOne)).Quo(lim, chPPM)
		}
		if found, pu := s.firstLevel(scale, cursor, blk.c); found {
			l := new(big.Int).Mul(pu, chPPM)
			var r *big.Int
			if s.bid {
				r = new(big.Int).Mul(s.P, chN(1_000_000+rp))
				if l.Cmp(r) > 0 {
					return chPlan{err: "Refused"}
				}
			} else {
				m := uint64(0)
				if 1_000_000 > rp {
					m = 1_000_000 - rp
				}
				r = new(big.Int).Mul(s.P, chN(m))
				if l.Cmp(r) < 0 {
					return chPlan{err: "Refused"}
				}
			}
		}
	} else if s.bid {
		lim.Set(chMax256)
	}
	// _bucket :132-148
	last := chU(b, 188, 48)
	room := chF(b, 92, 96)
	if blk.nowMs > last {
		add := new(big.Int).Mul(s.bRate, chN(blk.nowMs-last))
		room.Add(room, add.Quo(add, big.NewInt(1000)))
	}
	if room.Cmp(s.bBurst) > 0 {
		room = new(big.Int).Set(s.bBurst)
	}
	lr, maxIn := new(big.Int).Set(chCap128), new(big.Int)
	if mode != chSwap {
		lr, maxIn = s.pool.laneStateOf(s.bid)
	}
	var inB, outB *big.Int
	if s.bid {
		inB = amountIn
		if maxIn.Sign() != 0 && maxIn.Cmp(amountIn) < 0 {
			inB = maxIn
		}
		outB = chMinB(lr, room)
	} else {
		i := chMinB(amountIn, room)
		if maxIn.Sign() != 0 && maxIn.Cmp(i) < 0 {
			i = maxIn
		}
		inB, outB = i, lr
	}
	inB, outB = chMinB(inB, chCap128), chMinB(outB, chCap128)
	rungs := uint(10)
	if mode == chFillable {
		if sd := chU(s.s1, 184, 8); sd != 0 {
			rungs = uint(sd)
		}
	}
	x, n := s.walk(inB, outB, scale, cursor, lim, blk.c, rungs)
	D := chAskScale
	if s.bid {
		D = chE36
	}
	out := new(big.Int).Quo(n, D)
	leg := x
	if s.bid {
		leg = out
	}
	meets := leg.Cmp(s.minFill()) >= 0
	// the side's new board state :1517-1529
	debit := leg
	units := new(big.Int).Add(x, unit)
	units.Sub(units, chOne).Quo(units, unit)
	level := new(big.Int)
	if room.Cmp(debit) > 0 {
		level.Sub(room, debit)
	}
	tm := last
	if blk.nowMs > tm {
		tm = blk.nowMs
	}
	refBy := new(big.Int).Rsh(b, 247)
	if refilled {
		refBy = chN(chU(s.word, 72, 48) & 0x1ff)
	}
	nb := big.NewInt(1)
	nb.Or(nb, chShl(nonce, 4)).Or(nb, chShl(new(big.Int).Add(filled, units), 52)).Or(nb, chShl(level, 92))
	nb.Or(nb, chShl(chN(tm), 188)).Or(nb, chShl(chN((chU(b, 236, 11)+1)&0x7ff), 236)).Or(nb, chShl(refBy, 247))
	return chPlan{x: x, out: out, meets: meets, board: nb}
}

// quote is the door's quote: the front's quoteFrom past _quoteTail :512-530. nil where it answers 0 or reverts.
func (s *chSide) quote(amountIn *big.Int, blk chBlock) *big.Int {
	if amountIn.Sign() <= 0 {
		return nil
	}
	p := s.evaluate(chQuote, amountIn, blk)
	if p.err != "" || p.x.Cmp(amountIn) != 0 || p.out.Sign() == 0 || !p.meets {
		return nil
	}
	return p.out
}

// fillable is the door's quoteFillable of the most the front takes (_quoteFillable :586-610, no credit mode).
func (s *chSide) fillable(blk chBlock) (x, out *big.Int) {
	p := s.evaluate(chFillable, new(big.Int).Sub(chCap128, chOne), blk)
	if p.err != "" || !p.meets || p.out.Sign() == 0 {
		return new(big.Int), new(big.Int)
	}
	return p.x, p.out
}

// swap is _swapNamed :702-748 on a listing with no credit mode: the plan read with no pool room, the board written,
// the pool's fillFromFront, then the minimum fill. It moves the side and the pool only where it does not revert.
func (s *chSide) swap(amountIn *big.Int, blk chBlock) (*big.Int, string) {
	p := s.evaluate(chSwap, amountIn, blk)
	if p.err != "" {
		return nil, p.err
	}
	if p.x.Cmp(amountIn) != 0 || p.out.Sign() == 0 {
		return nil, "NotEnoughDepth"
	}
	fee36 := s.P
	if fee36.Sign() == 0 {
		fee36 = s.a
	}
	if e := s.pool.fill(s.bid, amountIn, p.out, fee36); e != "" {
		return nil, e
	}
	if !p.meets {
		// the pool's fill would be undone with the revert; the harness never continues a route past a revert
		return nil, "BelowMinimum"
	}
	s.board = p.board
	return p.out, ""
}
