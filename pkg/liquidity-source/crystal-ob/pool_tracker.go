package crystalob

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	market := common.HexToAddress(p.Address)
	var info struct{ Info MarketInfoRPC }
	var canonical common.Address
	var book PriceLevelsRPC
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: crystalABI, Target: t.config.RouterAddress, Method: "getMarket",
			Params: []any{market}}, []any{&info}).
		AddCall(&ethrpc.Call{ABI: crystalABI, Target: t.config.RouterAddress, Method: "getMarketByTokens",
			Params: []any{common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)}},
			[]any{&canonical}).
		AddCall(&ethrpc.Call{ABI: crystalABI, Target: t.config.RouterAddress, Method: "getPriceLevelsFromMid",
			Params: []any{market, levelsDistance, levelsInterval, levelsMax}},
					[]any{&book}).
		TryBlockAndAggregate() // getPriceLevelsFromMid reverts on bonding-curve launchpad markets
	if err != nil {
		return p, err
	} else if !resp.Result[0] || !resp.Result[1] {
		return p, ErrMarketInactive
	}

	var extra Extra
	reserves := entity.PoolReserves{"0", "0"}
	// Launchpad markets still on the bonding curve map to a placeholder and can't be swapped.
	if canonical == market {
		if !resp.Result[2] {
			return p, ErrMarketInactive
		}
		m := &info.Info
		extra.TakerFee, extra.MakerRebate = m.TakerFee.Uint64(), m.MakerRebate.Uint64()
		if extra.Bids, extra.Asks, err = t.resolveOrders(ctx, market, resp.BlockNumber, book.Bids,
			book.Asks); err != nil {
			return p, err
		}
		var rq, rb uint256.Int
		if m.IsAMMEnabled {
			extra.ReserveQ, extra.ReserveB = uint256.MustFromBig(m.ReserveQuote), uint256.MustFromBig(m.ReserveBase)
			rq.Set(extra.ReserveQ)
			rb.Set(extra.ReserveB)
		}
		reserves = entity.PoolReserves{sumSizes(&rq, extra.Bids).Dec(), sumSizes(&rb, extra.Asks).Dec()}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func sumSizes(acc *uint256.Int, levels []Level) *uint256.Int {
	for _, lvl := range levels {
		for _, size := range lvl[1:] {
			acc.Add(acc, size)
		}
	}
	return acc
}

// resolveOrders splits each aggregated level (price<<128 | size) into its resting orders, since
// _marketOrder rounds per order. Both sides share each multicall. A side is cut before any level
// whose orders can't be resolved.
func (t *PoolTracker) resolveOrders(ctx context.Context, market common.Address, block *big.Int,
	bidsPacked, asksPacked []byte) (bids, asks []Level, err error) {
	nb := len(bidsPacked) / 32
	packed := append(bidsPacked[:nb*32:nb*32], asksPacked...)
	n := len(packed) / 32
	if n == 0 {
		return nil, nil, nil
	}
	levels := make([]Level, n)
	totals := make([]uint256.Int, n)
	heads := make([]struct{ L PriceLevelRPC }, n) // single-tuple outputs unpack into the first field
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
	for i := range n {
		var word uint256.Int
		word.SetBytes32(packed[i*32 : i*32+32])
		price := new(uint256.Int).Rsh(&word, 128)
		totals[i].Set(word.And(&word, big256.UMaxU128))
		levels[i] = Level{price}
		req.AddCall(&ethrpc.Call{ABI: crystalABI, Target: t.config.RouterAddress, Method: "getPriceLevel",
			Params: []any{market, price.ToBig()}}, []any{&heads[i]})
	}
	if err = aggregatePinned(ctx, req); err != nil {
		return nil, nil, err
	}

	// Walk the FIFO list from fillNext; most levels hold a single order (fillNext == latest).
	next := make([]*big.Int, n)
	var sum uint256.Int
	for i := range n {
		if heads[i].L.FillNext.Cmp(heads[i].L.Latest) == 0 {
			levels[i] = append(levels[i], &totals[i])
		} else {
			next[i] = heads[i].L.FillNext
		}
	}
	for range maxOrderRounds {
		var pending []int
		orders := make([]struct{ Order OrderRPC }, n)
		req = t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block)
		for i := range n {
			if next[i] == nil {
				continue
			}
			pending = append(pending, i)
			call := &ethrpc.Call{ABI: crystalABI, Target: t.config.RouterAddress, Method: "getOrder",
				Params: []any{market, levels[i][0].ToBig(), next[i]}}
			if id := uint256.MustFromBig(next[i]); id.Gt(maskU41) { // cloid pointer: userId | cloid << 41
				var userId uint256.Int
				call.Method = "getOrderByCloid"
				call.Params = []any{userId.And(id, maskU41).ToBig(), id.Rsh(id, 41).ToBig()}
			}
			req.AddCall(call, []any{&orders[i]})
		}
		if len(pending) == 0 {
			break
		}
		if err = aggregatePinned(ctx, req); err != nil {
			return nil, nil, err
		}
		for _, i := range pending {
			o := &orders[i].Order
			if o.Size.Sign() > 0 {
				levels[i] = append(levels[i], uint256.MustFromBig(o.Size))
			}
			if next[i] = o.FillAfter; o.FillAfter.Sign() == 0 ||
				sumSizes(sum.Clear(), levels[i:i+1]).Cmp(&totals[i]) >= 0 {
				next[i] = nil
			}
		}
	}

	cut := func(lo, hi int) []Level {
		i := lo
		for i < hi && next[i] == nil && sumSizes(sum.Clear(), levels[i:i+1]).Eq(&totals[i]) {
			i++
		}
		if i < hi {
			logger.WithFields(logger.Fields{"dexID": t.config.DexID, "market": hexutil.Encode(market[:]),
				"price": levels[i][0].Dec(), "orders": len(levels[i]) - 1, "kept": i - lo, "fetched": hi - lo}).
				Warn("crystal-ob: book side cut at a level whose orders could not be fully walked")
		}
		return levels[lo:i:i]
	}
	return cut(0, nb), cut(nb, n), nil
}

// aggregatePinned retries block-pinned reads: Monad RPC backends can lag the block that the
// first read was served at by a few hundred ms and answer "Unknown block".
func aggregatePinned(ctx context.Context, req *ethrpc.Request) (err error) {
	for i := range 3 {
		if _, err = req.Aggregate(); err == nil || !strings.Contains(err.Error(), "nknown block") {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * 300 * time.Millisecond):
		}
	}
	return err
}
