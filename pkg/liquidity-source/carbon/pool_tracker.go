package carbon

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":   t.config.DexId,
		"address": p.Address,
	}).Info("start updating pool state")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var prevExtra Extra
	_ = json.Unmarshal([]byte(p.Extra), &prevExtra) // best-effort; zero value just forces a full scan

	token0 := common.HexToAddress(staticExtra.Token0)
	token1 := common.HexToAddress(staticExtra.Token1)

	tradingFeePpm, err := t.getTradingFee(ctx, token0, token1)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   t.config.DexId,
			"address": p.Address,
			"error":   err,
		}).Warn("failed to get trading fee, using default")
		tradingFeePpm = defaultTradingFeePpm
	}

	now := time.Now()
	fullScanDue := prevExtra.LastFullScanTime == 0 ||
		now.Sub(time.Unix(prevExtra.LastFullScanTime, 0)) > fullScanInterval

	var strategies []Strategy
	var blockNumber *big.Int
	var lastFullScanTime int64
	var strategyCount int64

	if fullScanDue {
		if strategies, strategyCount, blockNumber, err = t.fullScan(ctx, token0, token1); err != nil {
			logger.WithFields(logger.Fields{
				"dexId":   t.config.DexId,
				"address": p.Address,
				"error":   err,
			}).Error("failed to full-scan strategies")
			return p, err
		}
		strategies = applyDustFilter(strategies)
		lastFullScanTime = now.Unix()
	} else {
		if strategies, strategyCount, blockNumber, lastFullScanTime, err = t.incrementalScan(ctx, token0, token1, prevExtra); err != nil {
			logger.WithFields(logger.Fields{
				"dexId":   t.config.DexId,
				"address": p.Address,
				"error":   err,
			}).Error("failed to incrementally update strategies")
			return p, err
		}
	}

	reserve0, reserve1 := u256.New0(), u256.New0()
	for _, s := range strategies {
		if s.Orders[0].Y != nil {
			reserve0.Add(reserve0, s.Orders[0].Y)
		}
		if s.Orders[1].Y != nil {
			reserve1.Add(reserve1, s.Orders[1].Y)
		}
	}

	extra := Extra{
		Strategies:       strategies,
		TradingFeePpm:    tradingFeePpm,
		LastFullScanTime: lastFullScanTime,
		StrategyCount:    strategyCount,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{reserve0.String(), reserve1.String()}
	p.Timestamp = now.Unix()
	p.BlockNumber = blockNumber.Uint64()

	logger.WithFields(logger.Fields{
		"dexId":         t.config.DexId,
		"address":       p.Address,
		"numStrategies": len(strategies),
		"reserve0":      reserve0.String(),
		"reserve1":      reserve1.String(),
	}).Info("finished updating pool state")

	return p, nil
}

func (t *PoolTracker) getTradingFee(ctx context.Context, token0, token1 common.Address) (uint32, error) {
	var pairFee uint32
	var globalFee uint32

	if _, err := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    controllerABI,
			Target: t.config.Controller.String(),
			Method: "pairTradingFeePPM",
			Params: []any{token0, token1},
		}, []any{&pairFee}).
		AddCall(&ethrpc.Call{
			ABI:    controllerABI,
			Target: t.config.Controller.String(),
			Method: "tradingFeePPM"}, []any{&globalFee}).
		TryAggregate(); err != nil {
		return 0, err
	}

	if pairFee > 0 {
		return pairFee, nil
	}

	return globalFee, nil
}

// fullScan re-fetches every strategy for the pair from chain and returns the on-chain
// strategiesByPairCount alongside them - the caller stores that count (not len(strategies),
// since dust-filtered strategies get dropped) so incrementalScan can detect newly created
// strategies later by diffing counts instead of re-fetching everything every poll.
func (t *PoolTracker) fullScan(ctx context.Context, token0, token1 common.Address) ([]Strategy, int64, *big.Int, error) {
	count, blockNumber, err := t.getStrategyCount(ctx, token0, token1)
	if err != nil {
		return nil, 0, nil, err
	}
	if count == nil || count.Sign() == 0 {
		return nil, 0, blockNumber, nil
	}

	totalCount := int(count.Int64())
	strategies := make([]Strategy, 0, totalCount)
	for offset := 0; offset < totalCount; offset += maxStrategiesPerBatch {
		endIndex := min(offset+maxStrategiesPerBatch, totalCount)

		batch, err := t.fetchStrategiesBatch(ctx, token0, token1, offset, endIndex, blockNumber)
		if err != nil {
			return nil, 0, nil, err
		}

		strategies = append(strategies, batch...)
	}

	return strategies, count.Int64(), blockNumber, nil
}

// incrementalScan avoids a full strategiesByPair walk on every poll: it only pays for a cheap
// count check, a fetch of any newly created strategies (detected via that count), and a
// refresh of the strategies we're actually tracking (dust-filtered ones aren't stored at all),
// since those are the only ones whose liquidity matters for quoting between full scans.
func (t *PoolTracker) incrementalScan(
	ctx context.Context,
	token0, token1 common.Address,
	prevExtra Extra,
) ([]Strategy, int64, *big.Int, int64, error) {
	count, blockNumber, err := t.getStrategyCount(ctx, token0, token1)
	if err != nil {
		return nil, 0, nil, 0, err
	}
	if count == nil {
		count = big.NewInt(0)
	}

	if count.Int64() < prevExtra.StrategyCount {
		// a strategy was deleted on-chain; index positions may have shifted underneath us,
		// so fall back to a full re-scan rather than trying to diff against stale indices.
		strategies, strategyCount, blockNumber, err := t.fullScan(ctx, token0, token1)
		if err != nil {
			return nil, 0, nil, 0, err
		}
		strategies = applyDustFilter(strategies)
		return strategies, strategyCount, blockNumber, time.Now().Unix(), nil
	}

	strategies := prevExtra.Strategies

	if count.Int64() > prevExtra.StrategyCount {
		newStrategies, err := t.fetchStrategiesBatch(
			ctx, token0, token1, int(prevExtra.StrategyCount), int(count.Int64()), blockNumber)
		if err != nil {
			return nil, 0, nil, 0, err
		}
		newStrategies = applyDustFilter(newStrategies)
		strategies = append(append(make([]Strategy, 0, len(strategies)+len(newStrategies)), strategies...), newStrategies...)
	}

	ids := make([]*big.Int, len(strategies))
	for i, s := range strategies {
		ids[i] = s.Id
	}

	if len(ids) > 0 {
		refreshed, err := t.fetchStrategiesByIDs(ctx, token0, ids, blockNumber)
		if err != nil {
			return nil, 0, nil, 0, err
		}
		strategies = refreshed
	}

	return strategies, count.Int64(), blockNumber, prevExtra.LastFullScanTime, nil
}

func (t *PoolTracker) getStrategyCount(ctx context.Context, token0, token1 common.Address) (*big.Int, *big.Int, error) {
	var count *big.Int
	resp, err := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    controllerABI,
			Target: t.config.Controller.String(),
			Method: "strategiesByPairCount",
			Params: []any{token0, token1}}, []any{&count}).
		Aggregate()
	if err != nil {
		logger.Errorf("failed to get strategies by pair count: %v", err)
		return nil, nil, err
	}

	return count, resp.BlockNumber, nil
}

func (t *PoolTracker) fetchStrategiesBatch(
	ctx context.Context,
	token0, token1 common.Address,
	startIndex, endIndex int,
	blockNumber *big.Int,
) ([]Strategy, error) {
	var rawStrategies []StrategyByPairResp
	if _, err := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{
			ABI:    controllerABI,
			Target: t.config.Controller.String(),
			Method: "strategiesByPair",
			Params: []any{token0, token1, big.NewInt(int64(startIndex)), big.NewInt(int64(endIndex))},
		}, []any{&rawStrategies}).
		Aggregate(); err != nil {
		return nil, err
	}

	strategies := make([]Strategy, 0, len(rawStrategies))
	for _, raw := range rawStrategies {
		strategies = append(strategies, mapRawStrategy(raw, token0))
	}

	return strategies, nil
}

// fetchStrategiesByIDs refreshes a specific, already-known set of strategies (the ones not
// dust-filtered out) via the single-strategy getter, batched through multicall.
func (t *PoolTracker) fetchStrategiesByIDs(
	ctx context.Context,
	token0 common.Address,
	ids []*big.Int,
	blockNumber *big.Int,
) ([]Strategy, error) {
	strategies := make([]Strategy, 0, len(ids))
	for offset := 0; offset < len(ids); offset += maxStrategiesPerBatch {
		endIndex := min(offset+maxStrategiesPerBatch, len(ids))
		chunk := ids[offset:endIndex]

		raws := make([]StrategyByPairResp, len(chunk))
		req := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumber)
		for i, id := range chunk {
			req.AddCall(&ethrpc.Call{
				ABI:    controllerABI,
				Target: t.config.Controller.String(),
				Method: "strategy",
				Params: []any{id},
			}, []any{&raws[i]})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}

		for _, raw := range raws {
			strategies = append(strategies, mapRawStrategy(raw, token0))
		}
	}

	return strategies, nil
}

func mapRawStrategy(raw StrategyByPairResp, token0 common.Address) Strategy {
	i0, i1 := 0, 1
	if raw.Tokens[0] != token0 {
		i0, i1 = 1, 0
	}

	return Strategy{
		Id: raw.ID,
		Orders: [2]Order{
			{
				Y: uint256.MustFromBig(raw.Orders[i0].Y),
				Z: uint256.MustFromBig(raw.Orders[i0].Z),
				A: raw.Orders[i0].A,
				B: raw.Orders[i0].B,
			},
			{
				Y: uint256.MustFromBig(raw.Orders[i1].Y),
				Z: uint256.MustFromBig(raw.Orders[i1].Z),
				A: raw.Orders[i1].A,
				B: raw.Orders[i1].B,
			},
		},
	}
}

// applyDustFilter clears an order's Y/Z when, on its side (token0-order or token1-order across
// all strategies), it fails either the liquidity floor (>= 1% of the largest order) or the rate
// floor (>= 80% of the best-priced order), then compacts out any strategy left dust on both
// sides. Strategies kept for only one side still occupy a slot (with Id) so incrementalScan's
// count-diff invariant holds for those; only fully-dust ones are dropped, and StrategyCount -
// not len(Strategies) - accounts for those in the on-chain index space.
func applyDustFilter(strategies []Strategy) []Strategy {
	var maxY, maxLimit [2]*uint256.Int
	for i := range maxY {
		maxY[i] = u256.New0()
		maxLimit[i] = u256.New0()
	}

	for i := range strategies {
		for oi := range 2 {
			order := &strategies[i].Orders[oi]
			if order.Y == nil || order.Y.Sign() == 0 {
				continue
			}

			if order.Y.Gt(maxY[oi]) {
				maxY[oi] = order.Y.Clone()
			}

			var limit uint256.Int
			getLimit(&limit, order)
			if limit.Gt(maxLimit[oi]) {
				maxLimit[oi] = limit.Clone()
			}
		}
	}

	out := strategies[:0]
	for i := range strategies {
		for oi := range 2 {
			order := &strategies[i].Orders[oi]
			if order.Y == nil || order.Y.Sign() == 0 {
				*order = Order{}
				continue
			}

			var lhs, rhs uint256.Int
			lhs.Mul(order.Y, uHundred)
			rhs.Mul(maxY[oi], uDustLiquidityPct)
			passLiquidity := lhs.Cmp(&rhs) >= 0

			var limit uint256.Int
			getLimit(&limit, order)
			lhs.Mul(&limit, uHundred)
			rhs.Mul(maxLimit[oi], uDustRatePct)
			passRate := lhs.Cmp(&rhs) >= 0

			if !passLiquidity || !passRate {
				*order = Order{}
			}
		}

		if strategies[i].Orders[0] != (Order{}) || strategies[i].Orders[1] != (Order{}) {
			out = append(out, strategies[i])
		}
	}

	return out
}
