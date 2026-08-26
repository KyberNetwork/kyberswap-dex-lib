package prismprop

import (
	"context"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState calls getOrderBook once per direction (token0->token1 and
// token1->token0). Each call returns two independent maker order lists
// (side0, side1) for that direction -- see prism-prop discovery notes: both
// sides are denominated the same way (amountIn in the sell token, amountOut
// in the buy token), so they're pooled as one combined set of resting orders
// per direction rather than treated as bid/ask.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})
	l.Info("Start getting new state")

	token0, token1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	var res0, res1 getOrderBookResult
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: t.config.RouterAddress,
		Method: methodGetOrderBook,
		Params: []any{token0, token1},
	}, []any{&res0}).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: t.config.RouterAddress,
		Method: methodGetOrderBook,
		Params: []any{token1, token0},
	}, []any{&res1}).TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to aggregate RPC requests")
		return entity.Pool{}, err
	}

	levels0 := toLevels(res0.Book.Side0, res0.Book.Side1, p.Tokens[0].Decimals, p.Tokens[1].Decimals)
	levels1 := toLevels(res1.Book.Side0, res1.Book.Side1, p.Tokens[1].Decimals, p.Tokens[0].Decimals)

	extra := orderbook.Extra{LevelsFrom: [2][]orderbook.Level{levels0, levels1}}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserve0, reserve1 float64
	for _, lvl := range levels0 {
		reserve0 += lvl.Size()
	}
	for _, lvl := range levels1 {
		reserve1 += lvl.Size()
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		strconv.FormatFloat(reserve0*math.Pow10(int(p.Tokens[0].Decimals)), 'f', 0, 64),
		strconv.FormatFloat(reserve1*math.Pow10(int(p.Tokens[1].Decimals)), 'f', 0, 64),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	l.Info("Finish updating state of pool")
	return p, nil
}

// toLevels merges both maker sides of one getOrderBook direction into a
// single price-descending list, matching order-book package's greedy-walk
// assumption (best price consumed first). On-chain orders arrive in no
// particular price order (confirmed against a live getOrderBook call), so
// this sort is required, not cosmetic.
//
// order-book.NewPoolSimulatorWith takes minTrade from LevelsFrom[i][0].Size()
// (see order-book/pool_simulator.go), so the first level here must be a
// zero-size sentinel -- otherwise the best-priced real order's own size
// would wrongly become the minimum tradeable amount, rejecting smaller
// trades that later (worse-priced) orders could still fill. Same pattern as
// kuru-ob's pool_tracker.go ("first level == min trade == 0").
func toLevels(side0, side1 Side, decimalsIn, decimalsOut uint8) []orderbook.Level {
	orders := make([]Order, 0, len(side0.Orders)+len(side1.Orders))
	orders = append(orders, side0.Orders...)
	orders = append(orders, side1.Orders...)

	scaleIn, scaleOut := math.Pow10(int(decimalsIn)), math.Pow10(int(decimalsOut))
	levels := make([]orderbook.Level, 1, 1+len(orders)) // levels[0] == zero-size sentinel
	for _, o := range orders {
		amountIn, _ := new(big.Float).SetInt(o.AmountIn).Float64()
		amountOut, _ := new(big.Float).SetInt(o.AmountOut).Float64()
		if amountIn <= 0 {
			continue
		}
		size := amountIn / scaleIn
		price := (amountOut / scaleOut) / size
		levels = append(levels, orderbook.Level{size, price})
	}

	rest := levels[1:]
	sort.Slice(rest, func(i, j int) bool { return rest[i].Price() > rest[j].Price() })
	return levels
}
