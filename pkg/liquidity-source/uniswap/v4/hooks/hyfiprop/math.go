package hyfiprop

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Side is the raw 3-slot storage of one book side (HyFi.Side).
type Side struct {
	Slot0 uint256.Int `json:"s"`
	WordA uint256.Int `json:"a"`
	WordB uint256.Int `json:"b"`
}

// PairConfig mirrors HyFi.PairConfig.
type PairConfig struct {
	TickWidth    uint256.Int `json:"tw"` // quote-wei per base-wei, scaled by 1e24
	BaseLiqUnit  uint256.Int `json:"lu"` // base-wei per tick liquidity unit (uint88)
	FeePerSecond uint32      `json:"fs"`
	BaseIsC0     bool        `json:"b0"`
}

// book indices: HyFi.Book {bid, ask}. The bid side is walked when selling base.
const (
	sideBid = iota
	sideAsk
)

// tradeResult is the outcome of HyFi._priceTrade for an exact-input trade.
type tradeResult struct {
	amountOut, fee uint256.Int
	isSellingBase  bool
	cur            uint64      // walk pointer after the trade
	left           uint256.Int // amountLeft after the trade
}

// walker is HyFi.WalkParams plus the unpacked tick words.
type walker struct {
	slot0         [32]byte // big-endian slot0, for the 4 head ticks
	wordA, wordB  [32]byte
	tip, cur, end uint64
	left          uint256.Int
	liqUnit       uint256.Int
	tickWidth     uint256.Int
	isBid         bool
}

func newWalker(s *Side, cfg *PairConfig, isBid bool) *walker {
	var t uint256.Int
	w := &walker{
		slot0: s.Slot0.Bytes32(), wordA: s.WordA.Bytes32(), wordB: s.WordB.Bytes32(),
		tip: s.Slot0[0] & (1<<40 - 1), tickWidth: cfg.TickWidth, liqUnit: cfg.BaseLiqUnit, isBid: isBid,
	}
	w.cur = t.Rsh(&s.Slot0, 112)[0] & 0xFF
	w.end = t.Rsh(&s.Slot0, 120)[0] & 0xFF
	w.left.And(t.Rsh(&s.Slot0, 128), uMask96)
	return w
}

// avail is the base-wei liquidity of tick i (HyFi._avail).
func (w *walker) avail(i uint64, res *uint256.Int) {
	var unit byte
	switch {
	case i < headTicks:
		unit = w.slot0[3-i] // bits 224+8i
	case i < headTicks+wordTicks:
		unit = w.wordA[31-(i-headTicks)]
	default:
		unit = w.wordB[31-(i-headTicks-wordTicks)]
	}
	// unit < 2^8 and liqUnit < 2^88, so the product fits
	res.SetUint64(uint64(unit))
	res.Mul(res, &w.liqUnit)
}

// price is (tip -/+ i) * tickWidth. tip < 2^40 and tickWidth < 2^128, so the product fits.
func (w *walker) price(i uint64, res *uint256.Int) error {
	p := w.tip + i
	if w.isBid {
		if i > w.tip {
			return ErrOverflow // checked subtraction reverts on-chain
		}
		p = w.tip - i
	}
	res.SetUint64(p)
	res.Mul(res, &w.tickWidth)
	return nil
}

// next returns the liquidity and price of the tick the walk is on, or skip when the tick is empty.
func (w *walker) next(i, start uint64, avail, price *uint256.Int) (skip bool, err error) {
	if i > w.end || i >= numTicks {
		return false, ErrInsufficientLiquidity
	}
	if i == start && !w.left.IsZero() {
		avail.Set(&w.left) // partially consumed tick from a previous trade
	} else if w.avail(i, avail); avail.IsZero() {
		return true, nil
	}
	return false, w.price(i, price)
}

// walkBase consumes exactly target base-wei and returns the quote amount (HyFi._walkBase).
func (w *walker) walkBase(target *uint256.Int, roundUp bool) (quote uint256.Int, err error) {
	var remaining, avail, price, q, newLeft uint256.Int
	remaining.Set(target)
	i := w.cur
	start := i
	for !remaining.IsZero() {
		skip, err := w.next(i, start, &avail, &price)
		if err != nil {
			return quote, err
		} else if skip {
			i++
			continue
		}
		if remaining.Lt(&avail) {
			big256.MulDivRounding(&q, &remaining, &price, uScale, roundUp)
			newLeft.Sub(&avail, &remaining)
			remaining.Clear()
		} else {
			big256.MulDivRounding(&q, &avail, &price, uScale, roundUp)
			remaining.Sub(&remaining, &avail)
			i++
		}
		if _, overflow := quote.AddOverflow(&quote, &q); overflow {
			return quote, ErrOverflow
		}
	}
	w.cur, w.left = i, newLeft
	return quote, nil
}

// walkQuote consumes exactly target quote-wei and returns the base amount (HyFi._walkQuote).
// quoteIsInput: the trader pays quote, so full ticks cost rounded-up quote and partial base rounds down.
func (w *walker) walkQuote(target *uint256.Int, quoteIsInput bool) (base uint256.Int, err error) {
	var remaining, avail, price, availInQuote, basePart, newLeft uint256.Int
	remaining.Set(target)
	i := w.cur
	start := i
	for !remaining.IsZero() {
		skip, err := w.next(i, start, &avail, &price)
		if err != nil {
			return base, err
		} else if skip {
			i++
			continue
		}
		big256.MulDivRounding(&availInQuote, &avail, &price, uScale, quoteIsInput)
		if remaining.Lt(&availInQuote) {
			// remaining < ceil(avail*price/SCALE), so remaining*SCALE/price < avail
			big256.MulDivRounding(&basePart, &remaining, uScale, &price, !quoteIsInput)
			base.Add(&base, &basePart)
			if newLeft.Sub(&avail, &basePart); newLeft.IsZero() {
				i++ // rounding up can consume the tick exactly; 0 must mean "untouched tick"
			}
			remaining.Clear()
		} else {
			base.Add(&base, &avail)
			remaining.Sub(&remaining, &availInQuote)
			i++
		}
	}
	w.cur, w.left = i, newLeft
	return base, nil
}

// priceExactIn mirrors HyFi._priceTrade for exact input. now is the execution block timestamp.
func priceExactIn(book *[2]Side, cfg *PairConfig, zeroForOne bool, amountIn *uint256.Int,
	now uint64) (res tradeResult, err error) {
	if cfg.TickWidth.IsZero() {
		return res, ErrPairNotConfigured
	} else if amountIn.BitLen() > 255 { // toInt256 reverts
		return res, ErrOverflow
	}
	res.isSellingBase = zeroForOne == cfg.BaseIsC0
	side := &book[sideAsk]
	if res.isSellingBase {
		side = &book[sideBid]
	}
	w := newWalker(side, cfg, res.isSellingBase)

	// staleness fee in pips, per second since the book timestamp, capped at 100%
	var tmp uint256.Int
	feePips := uint64(0)
	if bookTs := tmp.Rsh(&side.Slot0, 40)[0] & 0xFFFFFFFF; now > bookTs {
		feePips = min((now-bookTs)*uint64(cfg.FeePerSecond), pips) // < 2^32 * 2^24
	}

	var gross uint256.Int
	if res.isSellingBase {
		gross, err = w.walkBase(amountIn, false)
	} else {
		gross, err = w.walkQuote(amountIn, true)
	}
	if err != nil {
		return res, err
	}
	big256.MulDivUp(&res.fee, &gross, tmp.SetUint64(feePips), uPips)
	res.amountOut.Sub(&gross, &res.fee)
	res.cur, res.left = w.cur, w.left
	return res, nil
}

// persistWalk writes the walk pointer into slot0 like HyFi._trade.
func persistWalk(s *Side, cur uint64, left *uint256.Int) {
	var t uint256.Int
	s.Slot0.And(&s.Slot0, t.Not(maskCurLeft))
	s.Slot0.Or(&s.Slot0, t.Lsh(t.SetUint64(cur), 112))
	s.Slot0.Or(&s.Slot0, t.Lsh(left, 128))
}
