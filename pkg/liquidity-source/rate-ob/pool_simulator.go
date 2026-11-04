package rateob

import (
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// expiryHorizon: an order whose deadline falls within this long after the snapshot may expire
// before the swap lands, and an expired order is evicted instead of filled, so the walk stops in
// front of it rather than quote a fill that may not happen.
const expiryHorizon = 5 * time.Minute

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

// NewPoolSimulator rejects a stale snapshot only when params.Opts.StaleCheck is set (route finding);
// the liquidity indexer passes false.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	entityPool := params.EntityPool
	if params.Opts.StaleCheck && time.Since(time.Unix(entityPool.Timestamp, 0)) > orderbook.MaxAge {
		return nil, orderbook.ErrLevelsTooOld
	}
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if len(entityPool.Tokens) != 2 || staticExtra.DecDiff == 0 || uint64(extra.TakerFee) >= denom.Uint64() ||
		uint64(extra.SpreadSell) >= denom.Uint64() {
		return nil, ErrInvalidExtra
	}
	if bp := extra.BandPool; bp != nil && (bp.Anchor.IsZero() || len(bp.Bands) == 0 || len(bp.Bands) > maxBands) {
		return nil, ErrInvalidExtra
	}
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

// CalcAmountOut quotes the engine's taker path: the book sweep, then (when the pair's band pool
// is tracked and the walk ends somewhere the engine provably ends too) PoolFallbackLib.route for
// the remainder. The quote is exactly what limitBuy / limitSell fills when sent the used input.
func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	idxIn, idxOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
	} else if amountIn.IsZero() {
		return nil, ErrZeroAmountOut
	}
	// tokens are [base, quote]: quote in is a buy (limitBuy walks the asks).
	isBuy := idxIn == 1
	r, err := p.sweep(&amountIn, isBuy)
	if err != nil {
		return nil, err
	}
	q, err := p.quoteFrom(r, &amountIn, isBuy)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: q.out.ToBig()},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: q.fee.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: q.remaining.ToBig()},
		Gas:                    q.gas,
		SwapInfo:               q.si,
	}, nil
}

type quoteResult struct {
	out, fee, remaining uint256.Int
	gas                 int64
	si                  SwapInfo
}

// quoteFrom turns a sweep into the quote: book and pool when the pool leg is possible and passes
// the engine's bound, otherwise book-only (sending only what the book uses, so nothing reaches
// the pool).
func (p *PoolSimulator) quoteFrom(r *sweepResult, amountIn *uint256.Int, isBuy bool) (*quoteResult, error) {
	if !r.remaining.IsZero() && r.walkOK && p.BandPool != nil {
		leg, err := p.poolLeg(r, amountIn, isBuy)
		if err != nil {
			return nil, err
		}
		if leg != nil {
			q := &quoteResult{gas: gasBase + gasMatch*int64(r.matches) + gasEviction*int64(r.walkEvictions) +
				gasPoolLeg + gasPoolBand*int64(len(leg.fills))}
			q.out.Add(&r.out, &leg.out)
			q.fee.Add(&r.fee, &leg.fee)
			q.remaining.Sub(&r.remaining, &leg.in)
			at := r.walkAt
			levels := p.side(isBuy)
			if at.level < len(levels) && at.order >= len(levels[at.level].Orders) && !levels[at.level].More {
				at = cursor{level: at.level + 1}
			}
			q.si = SwapInfo{
				IsBuy:        isBuy,
				LimitPrice:   leg.limitPrice.Dec(),
				N:            p.MaxMatches,
				PoolAmountIn: leg.in.Dec(),
				Levels:       at.level,
				Orders:       at.order,
				Head:         at.head,
				Matches:      r.matches,
				Fills:        leg.fills,
			}
			q.si.Lmp.Set(&leg.lmp)
			return q, nil
		}
	}

	if r.lastPrice == nil || r.out.IsZero() {
		return nil, ErrInsufficientLiquidity
	}
	// Only what the committed fills used is spent: whatever the walk did after the last fill
	// does not happen when the executor sends exactly the used amount.
	var used uint256.Int
	used.Sub(amountIn, &r.remaining)
	if ok, err := p.sizeOK(r.lastPrice, &used, isBuy); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrOrderSizeTooSmall
	}

	// A level whose every order the commit removed is deleted from the book.
	commit := r.commit
	levels := p.side(isBuy)
	if commit.order >= len(levels[commit.level].Orders) && !levels[commit.level].More {
		commit = cursor{level: commit.level + 1}
	}
	q := &quoteResult{gas: gasBase + gasMatch*int64(r.matches) + gasEviction*int64(r.evictions)}
	q.out.Set(&r.out)
	q.fee.Set(&r.fee)
	q.remaining.Set(&r.remaining)
	q.si = SwapInfo{
		IsBuy:      isBuy,
		LimitPrice: r.lastPrice.Dec(),
		N:          p.MaxMatches,
		Levels:     commit.level,
		Orders:     commit.order,
		Head:       commit.head,
		Matches:    r.matches,
	}
	q.si.Lmp.Set(r.lastPrice)
	return q, nil
}

// sizeOK is MatchingEngine._deposit's guard: the order's size, converted at its price, must be
// larger than one unit's.
func (p *PoolSimulator) sizeOK(price, amount *uint256.Int, isBuy bool) (bool, error) {
	var converted, minRequired, one uint256.Int
	if err := convert(&converted, price, amount, !isBuy, p.DecDiff, p.BaseBQuote); err != nil {
		return false, err
	}
	if err := convert(&minRequired, price, one.SetOne(), !isBuy, p.DecDiff, p.BaseBQuote); err != nil {
		return false, err
	}
	return converted.Gt(&minRequired), nil
}

func (p *PoolSimulator) side(isBuy bool) []Level {
	if isBuy {
		return p.Asks
	}
	return p.Bids
}

type sweepResult struct {
	// The committed fills: everything up to and including the last match.
	out, fee, remaining uint256.Int
	evictions, matches  int
	lastPrice           *uint256.Int
	commit              cursor

	// bound is the engine's reach: min(price, ceil(lmp*(D+s)/D)) for a buy,
	// max(price, floor(lmp*(D-s)/D)) for a sell.
	bound uint256.Int
	// Where the engine's walk ends when it is sent MORE than the book takes, so a remainder is
	// left after the last fill. walkOK means the engine stops exactly there too (the end of a
	// side read in full, the reach, MaxMatches, or a pass with no match); it is false when the
	// walk stopped in front of something it cannot predict. walkLmp is the lmp the walk leaves
	// (MatchingLib sets it to the price of every level it enters, matched or not), walkHead the
	// consumed side's head after it (0 if none), walkAt the walk's position and walkEvictions
	// every eviction it made, including any after the last fill.
	walkOK        bool
	walkLmp       uint256.Int
	walkHead      uint256.Int
	walkAt        cursor
	walkEvictions int
}

// cursor is a position in the consumed side: level index, order index, and the head order's
// deposit if a partial fill changed it.
type cursor struct {
	level, order int
	head         *uint256.Int
}

// sweep mirrors MatchingLib._limitOrder for a taker order that rests nothing (isMaker false),
// with Orderbook.fpop / execute / _sendFunds for each order.
//
// What it quotes is what the book fills: the walk stops at the engine's reach (the last matched
// price moved by the pair's limit spread), at MaxMatches, where the book runs out, and in front
// of anything it cannot predict (an order about to expire, or a level read only in part). With
// no pool leg, input left over is returned as RemainingTokenAmountIn; sending only the used amount
// makes the last fill end the sweep, so the engine never routes a remainder anywhere else.
func (p *PoolSimulator) sweep(amountIn *uint256.Int, isBuy bool) (*sweepResult, error) {
	levels, spread, full := p.Bids, p.SpreadSell, p.BidsFull
	if isBuy {
		levels, spread, full = p.Asks, p.SpreadBuy, p.AsksFull
	}
	if p.Lmp.IsZero() || p.MaxMatches == 0 {
		return nil, ErrInsufficientLiquidity
	}
	r := &sweepResult{}
	// The engine clamps the order's price to the last matched price moved by the limit spread:
	// limitBuy takes min(price, ceil(lmp*(D+s)/D)), limitSell max(price, floor(lmp*(D-s)/D)).
	bound := &r.bound
	var tmp uint256.Int
	if isBuy {
		if _, overflow := bound.MulOverflow(&p.Lmp, tmp.AddUint64(denom, uint64(spread))); overflow {
			return nil, ErrOverflow
		}
		bound.Add(bound, tmp.SubUint64(denom, 1)).Div(bound, denom)
	} else {
		if _, overflow := bound.MulOverflow(&p.Lmp, tmp.SubUint64(denom, uint64(spread))); overflow {
			return nil, ErrOverflow
		}
		bound.Div(bound, denom)
	}

	horizon := p.Timestamp + uint64(expiryHorizon/time.Second)
	fee := uint256.NewInt(uint64(p.TakerFee))
	var remaining, out, feeTotal, required, gross, f uint256.Int
	remaining.Set(amountIn)
	r.remaining.Set(amountIn)

	var at cursor // walk position
	var matches, evictions int
	var entered *uint256.Int // the last level the engine entered (MatchingLib sets lmp to it)
	n := int(p.MaxMatches)
	walkOK := false
	var walkHead *uint256.Int

walk:
	for !remaining.IsZero() {
		if at.level >= len(levels) {
			walkOK = !full
			break
		}
		level := &levels[at.level]
		price := &level.Price
		if isBuy && price.Gt(bound) || !isBuy && price.Lt(bound) || matches >= n {
			walkOK, walkHead = true, price
			break
		}
		entered = price
		prevMatches, roundEvictions := matches, 0
		// One MatchingLib.matchAt pass over the level's queue.
		for !remaining.IsZero() && at.order < len(level.Orders) && matches < n && roundEvictions < maxDustEvictions {
			order := &level.Orders[at.order]
			deposit := &order.Deposit
			if at.head != nil {
				deposit = at.head
			}
			if order.Deadline != 0 {
				if p.Timestamp > order.Deadline { // expired already: fpop evicts it
					at.order, at.head = at.order+1, nil
					roundEvictions, evictions = roundEvictions+1, evictions+1
					continue
				} else if order.Deadline <= horizon {
					break walk
				}
			}
			// fpop: what the taker must give to clear the order, in the taker's asset.
			if err := convert(&required, price, deposit, isBuy, p.DecDiff, p.BaseBQuote); err != nil {
				return nil, err
			}
			if required.IsZero() { // dust: fpop evicts it
				at.order, at.head = at.order+1, nil
				roundEvictions, evictions = roundEvictions+1, evictions+1
				continue
			}
			var newHead *uint256.Int
			removed := true
			if remaining.Gt(&required) { // clears the order: pay required, receive the whole deposit
				gross.Set(deposit)
				remaining.Sub(&remaining, &required)
			} else { // the last match: pay everything left
				gross.Set(deposit)
				if !remaining.Eq(&required) { // not a clear: _decreaseOrder
					if err := convert(&tmp, price, &remaining, !isBuy, p.DecDiff, p.BaseBQuote); err != nil {
						return nil, err
					}
					if tmp.Lt(deposit) {
						gross.Set(&tmp)
						newHead = new(uint256.Int).Sub(deposit, &tmp)
						// _evictIfDust: a remainder that can never fill is evicted at once.
						if err := convert(&tmp, price, newHead, isBuy, p.DecDiff, p.BaseBQuote); err != nil {
							return nil, err
						}
						removed = tmp.IsZero()
					}
				}
				remaining.Clear()
			}
			// _sendFunds: the taker fee is floored per match on what the taker receives.
			if _, overflow := f.MulOverflow(&gross, fee); overflow {
				return nil, ErrOverflow
			}
			f.Div(&f, denom)
			out.Add(&out, gross.Sub(&gross, &f))
			feeTotal.Add(&feeTotal, &f)
			matches++
			if removed {
				at.order, at.head = at.order+1, nil
			} else {
				at.head = newHead
			}
			r.commit, r.lastPrice = at, price
			r.out.Set(&out)
			r.fee.Set(&feeTotal)
			r.evictions = evictions
			r.remaining.Set(&remaining)
		}
		if remaining.IsZero() {
			break
		}
		if matches == prevMatches {
			// A pass with no match ends the sweep: MatchingLib sets the head to 0. Unless the pass
			// ran out of read orders on a level that holds more, which the engine would go on to.
			walkOK = at.order < len(level.Orders) || !level.More || roundEvictions >= maxDustEvictions
			break
		}
		if at.order >= len(level.Orders) {
			if level.More { // the rest of this level was not read
				if walkOK = matches >= n; walkOK {
					walkHead = price // the level keeps its unread orders
				}
				break
			}
			at = cursor{level: at.level + 1}
		}
	}

	r.matches = matches
	r.walkOK = walkOK
	r.walkAt = at
	r.walkEvictions = evictions
	if entered != nil {
		r.walkLmp.Set(entered)
	} else {
		r.walkLmp.Set(&p.Lmp)
	}
	if walkHead != nil {
		r.walkHead.Set(walkHead)
	}
	return r, nil
}

// convert mirrors Orderbook.convert: floor to the 1e8 price scale first, then scale by DecDiff.
// baseToQuote is Orderbook's isBid argument.
func convert(z, price, amount *uint256.Int, baseToQuote bool, decDiff uint64, baseBQuote bool) error {
	var d uint256.Int
	d.SetUint64(decDiff)
	if baseToQuote {
		if _, overflow := z.MulOverflow(amount, price); overflow {
			return ErrOverflow
		}
		z.Div(z, denom)
		if baseBQuote {
			z.Div(z, &d)
		} else if _, overflow := z.MulOverflow(z, &d); overflow {
			return ErrOverflow
		}
		return nil
	}
	if price.IsZero() {
		return ErrInvalidExtra
	}
	if _, overflow := z.MulOverflow(amount, denom); overflow {
		return ErrOverflow
	}
	z.Div(z, price)
	if baseBQuote {
		if _, overflow := z.MulOverflow(z, &d); overflow {
			return ErrOverflow
		}
	} else {
		z.Div(z, &d)
	}
	return nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	// UpdateBalance never writes through these: it replaces the slices, the level it changes and
	// the band pool (copy-on-write).
	cloned.Bids, cloned.Asks = slices.Clone(p.Bids), slices.Clone(p.Asks)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	side := &p.Bids
	if si.IsBuy {
		side = &p.Asks
	}
	levels := slices.Clone((*side)[min(si.Levels, len(*side)):])
	if len(levels) > 0 && (si.Orders > 0 || si.Head != nil) {
		orders := slices.Clone(levels[0].Orders[min(si.Orders, len(levels[0].Orders)):])
		if si.Head != nil && len(orders) > 0 {
			orders[0].Deposit.Set(si.Head)
		}
		levels[0].Orders = orders
		// A level a pass emptied by evictions is cleared by the next order (clearEmptyHead).
		if len(orders) == 0 && !levels[0].More {
			levels = levels[1:]
		}
	}
	*side = levels
	p.Lmp.Set(&si.Lmp)

	if len(si.Fills) > 0 && p.BandPool != nil {
		bp := *p.BandPool
		bp.Bands = slices.Clone(p.BandPool.Bands)
		for _, f := range si.Fills {
			b := &bp.Bands[f.band]
			if si.IsBuy { // quote in, base out
				b.BaseReserve.Sub(&b.BaseReserve, &f.out)
				b.QuoteReserve.Add(&b.QuoteReserve, &f.in)
			} else {
				b.QuoteReserve.Sub(&b.QuoteReserve, &f.out)
				b.BaseReserve.Add(&b.BaseReserve, &f.in)
			}
		}
		p.BandPool = &bp
	}
}

func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Engine
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{ApprovalAddress: p.Engine, BlockNumber: p.Info.BlockNumber}
}

// SwapReceiveNativeIn is false: limitBuy and limitSell are not payable, the input is always ERC-20.
func (p *PoolSimulator) SwapReceiveNativeIn(_, _ string, _ valueobject.ChainID) bool {
	return false
}

// SwapReturnNativeOut: the book pays a WETH leg as native coin (Orderbook._pay unwraps it and
// sends with a 2,300 gas stipend, paying WETH only if that send fails), and so does the pool leg
// (PoolFallbackLib has the pool pay the engine, which unwraps and sends the same way).
func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, _ valueobject.ChainID) bool {
	return p.NativeOut != "" && strings.EqualFold(tokenOut, p.NativeOut)
}
