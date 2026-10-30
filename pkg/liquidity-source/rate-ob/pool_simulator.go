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
	r, err := p.sweep(&amountIn, idxIn == 1)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: r.out.ToBig()},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: r.fee.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: r.remaining.ToBig()},
		Gas:                    gasBase + gasMatch*int64(r.si.Matches) + gasEviction*int64(r.evictions),
		SwapInfo:               r.si,
	}, nil
}

type sweepResult struct {
	out, fee, remaining uint256.Int
	evictions           int
	si                  SwapInfo
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
// of anything it cannot predict (an order about to expire, or a level read only in part). Input
// left over is returned as RemainingTokenAmountIn; sending only the used amount makes the last
// fill end the sweep, so the engine never routes a remainder anywhere else.
func (p *PoolSimulator) sweep(amountIn *uint256.Int, isBuy bool) (*sweepResult, error) {
	levels, spread := p.Bids, p.SpreadSell
	if isBuy {
		levels, spread = p.Asks, p.SpreadBuy
	}
	if p.Lmp.IsZero() || len(levels) == 0 || p.MaxMatches == 0 {
		return nil, ErrInsufficientLiquidity
	}
	// The engine clamps the order's price to the last matched price moved by the limit spread:
	// limitBuy takes min(price, ceil(lmp*(D+s)/D)), limitSell max(price, floor(lmp*(D-s)/D)).
	var bound, tmp uint256.Int
	if isBuy {
		if _, overflow := bound.MulOverflow(&p.Lmp, tmp.AddUint64(denom, uint64(spread))); overflow {
			return nil, ErrOverflow
		}
		bound.Add(&bound, tmp.SubUint64(denom, 1)).Div(&bound, denom)
	} else {
		if _, overflow := bound.MulOverflow(&p.Lmp, tmp.SubUint64(denom, uint64(spread))); overflow {
			return nil, ErrOverflow
		}
		bound.Div(&bound, denom)
	}

	horizon := p.Timestamp + uint64(expiryHorizon/time.Second)
	fee := uint256.NewInt(uint64(p.TakerFee))
	r := &sweepResult{}
	var remaining, out, feeTotal, required, gross, f uint256.Int
	remaining.Set(amountIn)

	var at cursor // walk position
	var commit cursor
	var committedOut, committedFee, committedRemaining uint256.Int
	var matches, evictions, committedEvictions int
	var lastPrice *uint256.Int
	n := int(p.MaxMatches)

walk:
	for !remaining.IsZero() && at.level < len(levels) && matches < n {
		level := &levels[at.level]
		price := &level.Price
		if isBuy && price.Gt(&bound) || !isBuy && price.Lt(&bound) {
			break
		}
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
			commit, lastPrice = at, price
			committedOut.Set(&out)
			committedFee.Set(&feeTotal)
			committedEvictions = evictions
			committedRemaining.Set(&remaining)
		}
		if matches == prevMatches { // a pass with no match ends the sweep (head set to 0)
			break
		}
		if at.order >= len(level.Orders) {
			if level.More { // the rest of this level was not read
				break
			}
			at = cursor{level: at.level + 1}
		}
	}

	if lastPrice == nil || committedOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}
	// Only what the committed fills used is spent: whatever the walk did after the last fill
	// does not happen when the executor sends exactly the used amount.
	var used uint256.Int
	r.remaining.Set(&committedRemaining)
	used.Sub(amountIn, &committedRemaining)

	// MatchingEngine._deposit rejects an order whose size converts to no more than one unit.
	var converted, minRequired uint256.Int
	if err := convert(&converted, lastPrice, &used, !isBuy, p.DecDiff, p.BaseBQuote); err != nil {
		return nil, err
	}
	if err := convert(&minRequired, lastPrice, tmp.SetOne(), !isBuy, p.DecDiff, p.BaseBQuote); err != nil {
		return nil, err
	}
	if !converted.Gt(&minRequired) {
		return nil, ErrOrderSizeTooSmall
	}

	// A level whose every order the commit removed is deleted from the book.
	if commit.order >= len(levels[commit.level].Orders) && !levels[commit.level].More {
		commit = cursor{level: commit.level + 1}
	}
	r.out.Set(&committedOut)
	r.fee.Set(&committedFee)
	r.evictions = committedEvictions
	r.si = SwapInfo{
		IsBuy:      isBuy,
		LimitPrice: lastPrice.Dec(),
		N:          p.MaxMatches,
		Levels:     commit.level,
		Orders:     commit.order,
		Head:       commit.head,
		Matches:    matches,
	}
	r.si.Lmp.Set(lastPrice)
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
	// UpdateBalance never writes through these: it replaces the slices and the level it changes.
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
	}
	*side = levels
	p.Lmp.Set(&si.Lmp)
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
// sends with a 2,300 gas stipend, paying WETH only if that send fails).
func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, _ valueobject.ChainID) bool {
	return p.NativeOut != "" && strings.EqualFold(tokenOut, p.NativeOut)
}
