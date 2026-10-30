package rateob

import (
	"context"
	"math/big"
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
// level's order queue and the block timestamp pinned to that same block.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	pair := common.HexToAddress(p.Address)
	base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	var lmp, listingDate *big.Int
	var bidPrices, askPrices []*big.Int
	var spread struct {
		Buy  uint32
		Sell uint32
	}
	var takerFee, maxMatches uint32
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
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for _, ok := range resp.Result[:numStateCalls] {
		if !ok {
			return p, ErrCallFailed
		}
	}

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
	if _, err := req.Aggregate(); err != nil {
		return p, err
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
		}
		extra.Lmp.SetFromBig(lmp)
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
