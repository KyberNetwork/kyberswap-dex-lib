package rateob

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

const numStateCalls = 7

// GetNewPoolState reads the pair's price levels and engine parameters in one multicall, then every
// level's order queue, the band pool's state and the block timestamp pinned to that same block.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	pair := common.HexToAddress(p.Address)
	base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
	engine := common.HexToAddress(t.config.Engine)

	var lmp, listingDate *big.Int
	var bidPrices, askPrices []*big.Int
	var spread struct {
		Buy  uint32
		Sell uint32
	}
	var takerFee, maxMatches uint32
	var pr poolReads
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "lmp"}, []any{&lmp}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "getPrices",
			Params: []any{true, uint32(bookDepth)}}, []any{&bidPrices}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "getPrices",
			Params: []any{false, uint32(bookDepth)}}, []any{&askPrices}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "lmtSpreadLimits",
			Params: []any{pair}}, []any{&spread}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "feeOf",
			Params: []any{base, quote, common.HexToAddress(t.config.Executor), false}}, []any{&takerFee}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "maxMatches"}, []any{&maxMatches}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "listingDates",
			Params: []any{pair}}, []any{&listingDate}).
		// The band pool leg; any of these failing leaves the quote book-only.
		AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "getPool"}, []any{&pr.pool}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "twap",
			Params: []any{uint32(1)}}, []any{&pr.newest}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "getSpread",
			Params: []any{pair, true, true}}, []any{&pr.railUp}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "getSpread",
			Params: []any{pair, false, true}}, []any{&pr.railDown}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "feeOf",
			Params: []any{base, quote, engine, false}}, []any{&pr.engineFee}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: t.config.Engine, Method: "getStopOrderEngine"},
			[]any{&pr.stopEngine}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for _, ok := range resp.Result[:numStateCalls] {
		if !ok {
			return p, ErrCallFailed
		}
	}
	pr.ok = !slices.Contains(resp.Result[numStateCalls:], false) && pr.pool != (common.Address{})

	bids, asks := nonZero(bidPrices), nonZero(askPrices)
	bidOrders, askOrders := make([][]rpcOrder, len(bids)), make([][]rpcOrder, len(asks))
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i, price := range bids {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "getOrders",
			Params: []any{true, price, uint32(ordersPerLevel)}}, []any{&bidOrders[i]})
	}
	for i, price := range asks {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: p.Address, Method: "getOrders",
			Params: []any{false, price, uint32(ordersPerLevel)}}, []any{&askOrders[i]})
	}
	numOrderCalls := len(bids) + len(asks)
	if pr.ok {
		pr.addCalls(req, pair)
	}
	orderResp, err := req.TryAggregate()
	if err != nil {
		return p, err
	}
	for _, ok := range orderResp.Result[:numOrderCalls] {
		if !ok {
			return p, ErrCallFailed
		}
	}
	if pr.ok {
		pr.ok = pr.checkCalls(orderResp.Result[numOrderCalls:])
	}
	if pr.ok && pr.stopBook != (common.Address{}) {
		// Any stop order resting on the pair may activate on the remainder's way to the pool
		// (StopOrderHandoffLib) and change the book under it, so then the pool is not quoted.
		var bidTrigger, askTrigger *big.Int
		if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{ABI: rateABI, Target: pr.stopBook.Hex(), Method: "triggerHead",
				Params: []any{true}}, []any{&bidTrigger}).
			AddCall(&ethrpc.Call{ABI: rateABI, Target: pr.stopBook.Hex(), Method: "triggerHead",
				Params: []any{false}}, []any{&askTrigger}).
			Aggregate(); err != nil {
			return p, err
		}
		pr.ok = bidTrigger.Sign() == 0 && askTrigger.Sign() == 0
	}
	ts, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber).
		GetCurrentBlockTimestamp()
	if err != nil {
		return p, err
	}
	timestamp := new(big.Int).SetUint64(ts)

	var extra Extra
	var baseReserve, quoteReserve uint256.Int
	// MatchingEngine._deposit reverts PairNotListedYet before the listing date, and with no last
	// matched price limitBuy's reach is zero: nothing can fill either way.
	if listingDate.Cmp(timestamp) <= 0 && lmp.Sign() > 0 {
		extra = Extra{
			SpreadBuy:  spread.Buy,
			SpreadSell: spread.Sell,
			TakerFee:   takerFee,
			MaxMatches: maxMatches,
			Timestamp:  timestamp.Uint64(),
			Bids:       toLevels(bids, bidOrders, &quoteReserve),
			Asks:       toLevels(asks, askOrders, &baseReserve),
			BidsFull:   len(bids) >= bookDepth,
			AsksFull:   len(asks) >= bookDepth,
		}
		extra.Lmp.SetFromBig(lmp)
		extra.BandPool = pr.bandPool(&extra.Lmp, &baseReserve, &quoteReserve)
	}
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{baseReserve.Dec(), quoteReserve.Dec()}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	return p, nil
}

// poolReads collects the band pool's state. Everything is read in the tracker's pinned block.
type poolReads struct {
	ok         bool
	pool       common.Address
	stopEngine common.Address
	stopBook   common.Address
	// newest is twap(1): the average since the newest oracle observation, over its age.
	newest struct {
		Price        *big.Int
		ActualWindow uint32
	}
	railUp, railDown, engineFee uint32
	anchor                      *big.Int
	limitBuy, limitSell         uint32
	bandCount                   *big.Int
	bands                       [maxBands]struct {
		SpreadFrac     uint32
		Shares         *big.Int
		FeeGrowthBase  *big.Int
		FeeGrowthQuote *big.Int
		Open           bool
	}
	reserves [maxBands]struct {
		BaseReserve  *big.Int
		QuoteReserve *big.Int
	}
	multipliers [maxBands]uint32
}

func (pr *poolReads) addCalls(req *ethrpc.Request, pair common.Address) {
	pool := pr.pool.Hex()
	req.AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "anchorPrice"}, []any{&pr.anchor}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "limitBuy"}, []any{&pr.limitBuy}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "limitSell"}, []any{&pr.limitSell}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "bandCount"}, []any{&pr.bandCount})
	if pr.stopEngine != (common.Address{}) {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: pr.stopEngine.Hex(), Method: "stopOrderbooks",
			Params: []any{pair}}, []any{&pr.stopBook})
	}
	// Every band up to MAX_BANDS; those past bandCount revert and are ignored.
	for i := range maxBands {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "bands", Params: []any{uint8(i)}},
			[]any{&pr.bands[i]}).
			AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "bandReserves", Params: []any{uint8(i)}},
				[]any{&pr.reserves[i]}).
			AddCall(&ethrpc.Call{ABI: rateABI, Target: pool, Method: "bandFeeMultiplier", Params: []any{uint8(i)}},
				[]any{&pr.multipliers[i]})
	}
}

// checkCalls requires the scalar reads and every band below bandCount to have succeeded.
func (pr *poolReads) checkCalls(results []bool) bool {
	scalars := 4
	if pr.stopEngine != (common.Address{}) {
		scalars++
	}
	if len(results) != scalars+3*maxBands || slices.Contains(results[:scalars], false) ||
		pr.bandCount == nil || pr.bandCount.Sign() == 0 || pr.bandCount.Cmp(big.NewInt(maxBands)) > 0 {
		return false
	}
	return !slices.Contains(results[scalars:scalars+3*int(pr.bandCount.Int64())], false)
}

// bandPool returns the pool's state when its leg can be quoted exactly, else nil (book-only), and
// adds its reserves to the pair's.
//
// The anchor is the hard part: BandPool prices every band off Orderbook.twap(300), which moves
// with block time while lmp differs from the prices in the window, so a snapshot of it drifts
// before the swap lands even if nothing trades. It is quoted only when it CANNOT move:
//
//   - twap(1) reads the newest observation's age as its window and, the price being constant
//     since then, returns lmp itself. Requiring that window >= 300 s and that price == lmp means
//     every price in effect over the last 300 s (and since) is lmp, so twap(300) is lmp at any
//     later time -- until lmp next changes. (If the newest observation was written in this very
//     block, twap(1) averages the segment before it instead, so the same test still requires
//     that segment to be at lmp and at least 300 s long.)
//   - It stays lmp for the whole swap transaction: the book leg's setLmp writes an observation
//     carrying the old lmp at the execution timestamp, and twap(300) taken in that same second
//     weights the new lmp by zero seconds. The same holds for later swaps in the route, so
//     UpdateBalance keeps Anchor.
//   - anchorPrice() (the pool's own view, through the seedPrice fallback) must agree.
//
// Anything that changes lmp before execution (a trade in another transaction) is ordinary state
// drift, the same as for the book, and the next snapshot will not pass the 300 s test until the
// window has fully rolled over. Pairs with resting stop orders are skipped (see above), and so is
// a pool whose anchor, limits or band reads do not come back.
func (pr *poolReads) bandPool(lmp, baseReserve, quoteReserve *uint256.Int) *BandPool {
	if !pr.ok || pr.newest.Price == nil || pr.anchor == nil || pr.newest.ActualWindow < anchorWindow {
		return nil
	}
	var newest, anchor uint256.Int
	if newest.SetFromBig(pr.newest.Price) || anchor.SetFromBig(pr.anchor) || !newest.Eq(lmp) || !anchor.Eq(lmp) ||
		uint64(pr.engineFee) >= denom.Uint64() {
		return nil
	}
	bp := &BandPool{LimitBuy: pr.limitBuy, LimitSell: pr.limitSell, EngineFee: pr.engineFee, RailUp: pr.railUp,
		RailDown: pr.railDown, Bands: make([]Band, pr.bandCount.Int64())}
	bp.Anchor.Set(lmp)
	var poolBase, poolQuote uint256.Int
	for i := range bp.Bands {
		b, r := &pr.bands[i], &pr.reserves[i]
		band := &bp.Bands[i]
		band.SpreadFrac, band.FeeMultiplier = b.SpreadFrac, pr.multipliers[i]
		band.HasShares = b.Shares != nil && b.Shares.Sign() > 0
		if r.BaseReserve == nil || r.QuoteReserve == nil || band.BaseReserve.SetFromBig(r.BaseReserve) ||
			band.QuoteReserve.SetFromBig(r.QuoteReserve) {
			return nil
		}
		poolBase.Add(&poolBase, &band.BaseReserve)
		poolQuote.Add(&poolQuote, &band.QuoteReserve)
	}
	baseReserve.Add(baseReserve, &poolBase)
	quoteReserve.Add(quoteReserve, &poolQuote)
	return bp
}

// nonZero keeps the prefix of getPrices before its zero padding.
func nonZero(prices []*big.Int) []*big.Int {
	for i, price := range prices {
		if price == nil || price.Sign() == 0 {
			return prices[:i]
		}
	}
	return prices
}

// toLevels drops empty levels (a price can stay listed after its last order left) and adds every
// deposit to reserve. A level whose read filled every slot may hold more orders.
func toLevels(prices []*big.Int, orders [][]rpcOrder, reserve *uint256.Int) []Level {
	levels := make([]Level, 0, len(prices))
	for i, price := range prices {
		level := Level{Orders: make([]Order, 0, len(orders[i]))}
		if level.Price.SetFromBig(price) {
			break
		}
		for _, o := range orders[i] {
			if o.Owner == (common.Address{}) || o.DepositAmount == nil || o.DepositAmount.Sign() == 0 {
				break
			}
			var order Order
			if order.Deposit.SetFromBig(o.DepositAmount) {
				break
			}
			order.Deadline = o.Deadline
			reserve.Add(reserve, &order.Deposit)
			level.Orders = append(level.Orders, order)
		}
		if len(level.Orders) == 0 {
			continue
		}
		level.More = len(orders[i]) >= ordersPerLevel && len(level.Orders) == len(orders[i])
		levels = append(levels, level)
	}
	return levels
}
