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

	strategies, blockNumber, err := t.getStrategiesByPair(ctx, token0, token1)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   t.config.DexId,
			"address": p.Address,
			"error":   err,
		}).Error("failed to get strategies")
		return p, err
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
		Strategies:    strategies,
		TradingFeePpm: tradingFeePpm,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{reserve0.String(), reserve1.String()}
	p.Timestamp = time.Now().Unix()
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

func (t *PoolTracker) getStrategiesByPair(ctx context.Context, token0, token1 common.Address) ([]Strategy, *big.Int, error) {
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

	if count == nil || count.Sign() == 0 {
		return nil, resp.BlockNumber, nil
	}

	totalCount := int(count.Int64())
	strategies := make([]Strategy, 0, totalCount)
	for offset := 0; offset < totalCount; offset += maxStrategiesPerBatch {
		endIndex := offset + maxStrategiesPerBatch
		if endIndex > totalCount {
			endIndex = totalCount
		}

		batchStrategies, err := t.fetchStrategiesBatch(ctx, token0, token1, offset, endIndex, resp.BlockNumber)
		if err != nil {
			return nil, nil, err
		}

		strategies = append(strategies, batchStrategies...)
	}

	return strategies, resp.BlockNumber, nil
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
		// Skip no-liquidity strategies
		if raw.Orders[0].Y.Sign() == 0 && raw.Orders[1].Y.Sign() == 0 {
			continue
		}

		i0, i1 := 0, 1
		if raw.Tokens[0] != token0 {
			i0, i1 = 1, 0
		}

		strategies = append(strategies, Strategy{
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
		})
	}

	return strategies, nil
}
