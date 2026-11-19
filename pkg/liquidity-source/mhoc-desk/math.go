package mhocdesk

import "github.com/holiman/uint256"

// QuoteInput is one call of PropAMM._compute: the pool constants, the snapshot, the outgoing token's balance and
// floor, and the block the swap executes in.
type QuoteInput struct {
	D         *uint256.Int // 10^baseDecimals * PRICE_SCALE
	QuoteUnit *uint256.Int // 10^quoteDecimals
	KScale    uint64       // K_SCALE: the unit of the card's kBid and kAsk fields

	Extra       *Extra
	SellBase    bool         // tokenIn == BASE (taker sells BASE for QUOTE); false: tokenIn == QUOTE
	Unknown     bool         // tokenIn is neither (only reachable in tests: the router never asks)
	AmountIn    *uint256.Int //
	BalanceOut  *uint256.Int // pool balance of the outgoing token
	BlockNumber uint64       // block.number of the swap
	Now         uint64       // block.timestamp of the swap
	// WindowNow is the block.timestamp used for the per-minute budget; 0 = Now. The simulator sets it to the
	// earliest time the swap can land (the snapshot's block.timestamp + 1), so a minute that the wall clock has
	// crossed but the chain may not have does not free the budget: the quote can only be stricter than the pool.
	WindowNow uint64
}

// QuoteResult is the swap's output, its notional Q, and the counters the swap writes.
type QuoteResult struct {
	AmountOut *uint256.Int
	Notional  *uint256.Int
	Counters  Counters
}

// Quote ports PropAMM._compute line by line, including its order of checks and every rounding. With F the
// cumulative fill (max of the per-card and the per-block counter), k = kBid*K_SCALE on a sell and kAsk*K_SCALE on a
// buy, kw(x) = ceil(k*x*1e6/QU) and WAD = 1e18:
//
//	sell BASE (a):  Q = ceil(a*p/D);  f = dBid*1e12 + kw(2F + Q);  out = floor(a*p*(WAD-f) / (D*WAD))
//	buy BASE  (b):  Q = b;  A = WAD + dAsk*1e12;  out = floor(b*D*WAD / (p*(A + kw(F + Q))))
//	                then, if kw(F) != 0, out = floor(out*A / (A + kw(F)))
//
// With F = 0 (a fresh card in a block without earlier fills) both are the single-swap formulas. Quote returns
// exactly what quote()/swap() return in the same state and block, or the error swap reverts with.
func Quote(in QuoteInput) (*QuoteResult, error) {
	c := decodeCard(in.Extra.StateWord)
	if c.paused {
		return nil, ErrPoolPaused
	}
	if !c.live(in.BlockNumber, in.Now) {
		return nil, ErrStalePrice
	}
	if in.AmountIn.IsZero() {
		return nil, ErrZeroAmount
	}
	if in.AmountIn.Gt(maxAmountIn) {
		return nil, ErrExceedsMaxSize
	}

	// cap = min(qMax, maxNotional) * QUOTE_UNIT
	capWhole := c.qMax
	if in.Extra.MaxNotional < capWhole {
		capWhole = in.Extra.MaxNotional
	}
	capRaw := new(uint256.Int).Mul(uint256.NewInt(capWhole), in.QuoteUnit) // < 2^24 * 1e18

	if in.Unknown {
		return nil, ErrUnknownToken
	}
	p := c.price
	var q, num uint256.Int // q: this swap's notional in QUOTE raw units; num = a*p on a sell
	if in.SellBase {
		num.Mul(in.AmountIn, p) // <= 2^128 * 2^64
		ceilDiv(&q, &num, in.D)
	} else {
		q.Set(in.AmountIn)
	}
	if q.Gt(capRaw) {
		return nil, ErrExceedsMaxSize
	}

	// Per-card (Fs) and per-block (Fb) budgets, each against C, then the per-minute budget against W.
	cnt := &in.Extra.Counters
	var limC, used uint256.Int
	limC.Mul(uint256.NewInt(in.Extra.MaxPerUpdate), in.QuoteUnit)
	fs := new(uint256.Int)
	if cnt.CardSeq == c.seq && cnt.CardFilled != nil {
		fs.Set(cnt.CardFilled)
	}
	if _, of := used.AddOverflow(fs, &q); of || used.Gt(&limC) {
		return nil, ErrExceedsUpdateCap
	}
	wnow := in.WindowNow
	if wnow == 0 {
		wnow = in.Now
	}
	id := wnow / window
	var wused, bused uint256.Int
	wused.Set(&q)
	bused.Set(&q)
	if uint64(cnt.WindowID) == id {
		if cnt.WindowFilled != nil {
			wused.Add(&wused, cnt.WindowFilled)
		}
		if cnt.FillBlock == uint32(in.BlockNumber) && cnt.BlockFilled != nil {
			bused.Add(&bused, cnt.BlockFilled)
		}
	}
	if bused.Gt(&limC) {
		return nil, ErrExceedsUpdateCap
	}
	var limW uint256.Int
	limW.Mul(uint256.NewInt(uint64(in.Extra.MaxPerWindow)), in.QuoteUnit)
	if wused.Gt(&limW) { // W = 0: no trading
		return nil, ErrExceedsWindowCap
	}
	filled := fs // F = max(Fs, Fb)
	if fb := new(uint256.Int).Sub(&bused, &q); fb.Gt(filled) {
		filled = fb
	}

	out, err := amountOut(&c, in.KScale, in.SellBase, in.AmountIn, &num, &q, filled, in.D, in.QuoteUnit)
	if err != nil {
		return nil, err
	}
	if out.IsZero() {
		return nil, ErrNoFill
	}

	// inventory floor of the outgoing token: balance >= out + floor
	floor := in.Extra.FloorQuote
	if !in.SellBase {
		floor = in.Extra.FloorBase
	}
	var need uint256.Int
	if _, of := need.AddOverflow(out, floor); of || in.BalanceOut.Lt(&need) {
		return nil, ErrBelowFloor
	}
	return &QuoteResult{
		AmountOut: out,
		Notional:  &q,
		Counters:  afterFill(cnt, c.seq, &q, in.BlockNumber, wnow),
	}, nil
}

// amountOut is the pricing step of _compute, given the swap's notional q (num = a*p on a sell) and the cumulative
// fill F: the size slippage on F with the side's own coefficient (kBid*K_SCALE on a sell, kAsk*K_SCALE on a buy),
// the two half-spreads, and every rounding. It returns ErrNoFill for f >= 1 on a sell and for p = 0 on a buy; the
// caller rejects a zero output.
func amountOut(c *cardFields, kScale uint64, sellBase bool, amountIn, num, q, filled, d, quoteUnit *uint256.Int) (
	*uint256.Int, error) {
	p := c.price
	out := new(uint256.Int)
	if sellBase {
		// f = dBid + k*(2F + Q), k = kBid * K_SCALE (< 2^16 * 2^8)
		k := c.kBid * kScale
		var x uint256.Int
		x.Lsh(filled, 1).Add(&x, q)
		kw, err := kWad(k, &x, quoteUnit)
		if err != nil {
			return nil, err
		}
		f := new(uint256.Int).Mul(uint256.NewInt(c.dBid), spreadToWad)
		f.Add(f, kw)
		if !f.Lt(wad) {
			return nil, ErrNoFill
		}
		// num * (WAD - f) / (D * WAD)
		var t, den uint256.Int
		t.Sub(wad, f)
		if _, of := t.MulOverflow(num, &t); of {
			return nil, ErrOverflow
		}
		den.Mul(d, wad) // <= 1e48
		out.Div(&t, &den)
	} else {
		// out = G(F + Q) - G(F), G(x) = x*D / (p*(1 + dAsk + k*x)), k = kAsk * K_SCALE
		if p.IsZero() {
			return nil, ErrNoFill
		}
		k := c.kAsk * kScale
		a := new(uint256.Int).Mul(uint256.NewInt(c.dAsk), spreadToWad)
		a.Add(a, wad) // 1 + dAsk, exact
		var x uint256.Int
		x.Add(filled, q)
		kw, err := kWad(k, &x, quoteUnit)
		if err != nil {
			return nil, err
		}
		var n, den uint256.Int
		if _, of := n.MulOverflow(amountIn, d); of {
			return nil, ErrOverflow
		}
		if _, of := n.MulOverflow(&n, wad); of {
			return nil, ErrOverflow
		}
		den.Add(a, kw)
		if _, of := den.MulOverflow(p, &den); of {
			return nil, ErrOverflow
		}
		out.Div(&n, &den)
		// times (1 + dAsk) / (1 + dAsk + k*F), rounded down; skipped when k*F rounds to 0
		kf, err := kWad(k, filled, quoteUnit)
		if err != nil {
			return nil, err
		}
		if !kf.IsZero() {
			if _, of := out.MulOverflow(out, a); of {
				return nil, ErrOverflow
			}
			out.Div(out, den.Add(a, kf))
		}
	}
	return out, nil
}

// afterFill is what a fill of q writes, as _compute's (budget, window) result: the per-card counter restarts when
// the card changed, the per-minute counter when the minute changed, and the per-block counter when the block or
// the minute changed. Every field is a fresh value.
func afterFill(c *Counters, seq uint32, q *uint256.Int, blockNumber, now uint64) Counters {
	id := now / window
	n := Counters{
		CardSeq:      seq,
		CardFilled:   new(uint256.Int).Set(q),
		FillBlock:    uint32(blockNumber),
		BlockFilled:  new(uint256.Int).Set(q),
		WindowID:     uint32(id),
		WindowFilled: new(uint256.Int).Set(q),
	}
	if c.CardSeq == seq && c.CardFilled != nil {
		n.CardFilled.Add(n.CardFilled, c.CardFilled)
	}
	if uint64(c.WindowID) == id {
		if c.WindowFilled != nil {
			n.WindowFilled.Add(n.WindowFilled, c.WindowFilled)
		}
		if c.FillBlock == uint32(blockNumber) && c.BlockFilled != nil {
			n.BlockFilled.Add(n.BlockFilled, c.BlockFilled)
		}
	}
	return n
}

// kWad is _kWad: ceil(k*x*1e6 / QU), the size slippage in 1e18 fixed point, rounded up.
func kWad(k uint64, x, quoteUnit *uint256.Int) (*uint256.Int, error) {
	var kx uint256.Int
	if _, of := kx.MulOverflow(uint256.NewInt(k), x); of {
		return nil, ErrOverflow
	}
	if _, of := kx.MulOverflow(&kx, kToWad); of {
		return nil, ErrOverflow
	}
	return ceilDiv(new(uint256.Int), &kx, quoteUnit), nil
}

// ceilDiv is _ceilDiv: a == 0 ? 0 : (a - 1) / b + 1.
func ceilDiv(z, a, b *uint256.Int) *uint256.Int {
	if a.IsZero() {
		return z.Clear()
	}
	z.SubUint64(a, 1)
	z.Div(z, b)
	return z.AddUint64(z, 1)
}
