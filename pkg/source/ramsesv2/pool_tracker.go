package ramsesv2

import (
	"context"
	"maps"
	"math/big"
	"slices"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG(DexTypeRamsesV2, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG(DexTypeRamsesV2, NewPoolTracker)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {

	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool: %v", d.config.DexID, p.Address)

	var (
		rpcData   *FetchRPCResult
		poolTicks []ticklens.TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.FetchRPCData(ctx, &p, 0)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to fetch data from RPC")
		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		if d.config.AlwaysUseTickLens {
			poolTicks, err = ticklens.GetPoolTicksFromSC(ctx, d.ethrpcClient, d.config.TickLensAddress, p, nil)
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

		poolTicks, err = d.getPoolTicks(ctx, p.Address)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to query subgraph for pool ticks")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to fetch pool state, pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	var ticks []Tick
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to transform tickResp to tick")
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcData.Liquidity,
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		FeeTier:      rpcData.FeeTier,
		TickSpacing:  rpcData.TickSpacing,
		Tick:         rpcData.Slot0.Tick,
		Ticks:        ticks,
		Unlocked:     rpcData.Slot0.Unlocked,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = rpcData.BlockNumber
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}

	logger.Infof("[%s] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	var (
		liquidity   *big.Int
		slot0       Slot0
		feeTier     *big.Int
		tickSpacing *big.Int
		reserve0    = zeroBI
		reserve1    = zeroBI
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	if d.config.IsPoolV3 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    poolV3ABI,
			Target: p.Address,
			Method: methodV3Fee,
		}, []any{&feeTier})
	} else {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    poolV2ABI,
			Target: p.Address,
			Method: methodV2CurrentFee,
		}, []any{&feeTier})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolV2ABI,
		Target: p.Address,
		Method: methodV2GetLiquidity,
	}, []any{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolV2ABI,
		Target: p.Address,
		Method: methodV2GetSlot0,
	}, []any{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolV2ABI,
		Target: p.Address,
		Method: methodV2TickSpacing,
	}, []any{&tickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[1].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve1})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process Aggregate")
		return nil, err
	}

	return &FetchRPCResult{
		Liquidity:   liquidity,
		Slot0:       slot0,
		FeeTier:     feeTier.Int64(),
		TickSpacing: tickSpacing.Uint64(),
		Reserve0:    reserve0,
		Reserve1:    reserve1,
		BlockNumber: resp.BlockNumber.Uint64(),
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]ticklens.TickResp, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []ticklens.TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []ticklens.TickResp       `json:"ticks"`
			Meta  *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Ticks == nil {
					logger.WithFields(logger.Fields{
						"poolAddress":        poolAddress,
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Errorf("failed to query subgraph")

					return nil, err
				}
			} else {
				logger.WithFields(logger.Fields{
					"poolAddress":        poolAddress,
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Errorf("failed to query subgraph")

				return nil, err
			}
		}

		resp.Meta.CheckIsLagging(d.config.DexID, poolAddress)

		if len(resp.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Ticks...)

		if len(resp.Ticks) < graphFirstLimit {
			break
		}

		lastTickIdx = resp.Ticks[len(resp.Ticks)-1].TickIdx
	}

	return ticks, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"address": p.Address, "exchange": p.Exchange})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// Extract current ticks from entity pool extra
	var extra Extra
	if len(p.Extra) > 0 {
		err := json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return p, err
		}
	}

	ticks := map[int]struct{}{}
	for _, tick := range extra.Ticks {
		ticks[tick.Index] = struct{}{}
	}

	ticksToRefetch := make([]int, 0, len(ticks))
	for tickIdx := range ticks {
		ticksToRefetch = append(ticksToRefetch, tickIdx)
	}

	if len(ticksToRefetch) == 0 {
		return p, nil
	}

	refetchedTicks, err := t.queryRPCTicksByIndexes(ctx, p.Address, ticksToRefetch, p.BlockNumber)
	if err != nil {
		return p, err
	}

	// convert back to ramsesv2 ticks
	entityPoolTicks := make([]Tick, 0, len(refetchedTicks))
	for _, tick := range refetchedTicks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

	extra.Ticks = entityPoolTicks

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) newTicksBasedPool(
	ctx context.Context,
	p entity.Pool,
	logs []ethtypes.Log,
) (tickspkg.TicksBasedPool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to transform entity pool to ticks based pool")
		return ticksBasedPool, err
	}

	ticks, err := t.fetchTicksFromLogs(ctx, p, logs)
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to FetchTicksFromLogs")
		return ticksBasedPool, err
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)
	ticksBasedPool.BlockNumber = blockNumber

	if len(ticks) == 0 {
		return ticksBasedPool, nil
	}

	if err := tickspkg.ValidatePoolTicks(ticksBasedPool, ticks); err != nil {
		l.WithFields(logger.Fields{"numTicks": len(ticks), "error": err}).
			Warn("invalid pool ticks data after fetching ticks from logs")

		l.WithFields(logger.Fields{"numTicks": len(ticksBasedPool.Ticks)}).Info("fetch all ticks for pool")

		ticks, err = t.fetchAllTicksForPool(ctx, ticksBasedPool, ticks)
		if err != nil {
			l.WithFields(logger.Fields{"error": err}).Error("failed to fetch all ticks")

			return ticksBasedPool, err
		}

		if err := tickspkg.ValidateAllPoolTicks(ticksBasedPool, ticks); err != nil {
			l.WithFields(logger.Fields{"numTicks": len(ticks), "error": err}).
				Warnf("invalid pool ticks data after fetching all ticks stored in pool")
		}
	}

	for _, tick := range ticks {
		ticksBasedPool.Ticks[tick.TickIdx] = tick
	}

	return ticksBasedPool, nil
}

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool) (entity.Pool,
	error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	blockNumber := ticksBasedPool.BlockNumber

	rpcState, err := t.FetchRPCData(ctx, &p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, &p, 0)
			if err != nil {
				l.WithFields(logger.Fields{"error": err}).Error("failed to fetch latest state from RPC")
				return p, err
			}
		} else {
			l.WithFields(logger.Fields{"error": err, "blockNumber": blockNumber}).
				Error("failed to fetch state from RPC")
			return p, err
		}
	}

	entityPoolTicks := make([]Tick, 0, len(ticksBasedPool.Ticks))
	for _, tick := range ticksBasedPool.Ticks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcState.Liquidity,
		SqrtPriceX96: rpcState.Slot0.SqrtPriceX96,
		FeeTier:      rpcState.FeeTier,
		TickSpacing:  rpcState.TickSpacing,
		Tick:         rpcState.Slot0.Tick,
		Ticks:        entityPoolTicks,
		Unlocked:     rpcState.Slot0.Unlocked,
	})
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.SwapFee = float64(rpcState.FeeTier)
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcState.Reserve0.String(),
		rpcState.Reserve1.String(),
	}

	return p, err
}

func (t *PoolTracker) fetchAllTicksForPool(
	ctx context.Context,
	pool tickspkg.TicksBasedPool,
	ticksFromLogs []tickspkg.Tick,
) ([]tickspkg.Tick, error) {
	isTickFromLogs := map[int]struct{}{}
	lo.ForEach(ticksFromLogs, func(item tickspkg.Tick, index int) {
		isTickFromLogs[item.TickIdx] = struct{}{}
	})

	tickIdsFromPool := make([]int, 0, len(pool.Ticks))
	for tickIdx := range pool.Ticks {
		if _, ok := isTickFromLogs[tickIdx]; !ok {
			tickIdsFromPool = append(tickIdsFromPool, tickIdx)
		}
	}

	ticksFromPool, err := t.queryRPCTicksByIndexes(ctx, pool.Address, tickIdsFromPool, pool.BlockNumber)
	if err != nil {
		return nil, err
	}

	ticksMap := make(map[int]tickspkg.Tick)
	for _, tick := range ticksFromPool {
		ticksMap[tick.TickIdx] = tick
	}
	for _, tick := range ticksFromLogs {
		ticksMap[tick.TickIdx] = tick
	}

	return lo.Values(ticksMap), nil
}

func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context, pool entity.Pool, logs []ethtypes.Log,
) ([]tickspkg.Tick, error) {
	l := logger.WithFields(logger.Fields{"address": pool.Address, "exchange": pool.Exchange})

	if len(logs) == 0 {
		return nil, nil
	}

	tickIndexes, err := t.getTickIndexesFromLogs(logs)
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to getTickIndexesFromEvents")
		return nil, err
	}

	if len(tickIndexes) == 0 {
		return nil, nil
	}
	blockNumber := eth.GetBlockNumberFromLogs(logs)

	return t.queryRPCTicksByIndexes(ctx, pool.Address, tickIndexes, blockNumber)
}

func (t *PoolTracker) queryRPCTicksByIndexes(
	ctx context.Context, address string, tickIndexes []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		if t.config.IsPoolV3 {
			return t.queryRPCTicksV3ByChunk(ctx, address, tickIndexes, blockNumber)
		}

		return t.queryRPCTicksV2ByChunk(ctx, address, tickIndexes, blockNumber)
	}

	var (
		totalTicks = len(tickIndexes)
		ticks      = make([]tickspkg.Tick, 0, totalTicks)

		newTicks []tickspkg.Tick
		err      error
	)
	for i := 0; i < totalTicks; i += tickChunkSize {
		toIdx := i + tickChunkSize
		if toIdx > totalTicks {
			toIdx = totalTicks
		}

		if t.config.IsPoolV3 {
			newTicks, err = t.queryRPCTicksV3ByChunk(ctx, address, tickIndexes[i:toIdx], blockNumber)
		} else {
			newTicks, err = t.queryRPCTicksV2ByChunk(ctx, address, tickIndexes[i:toIdx], blockNumber)
		}
		if err != nil {
			return nil, err
		}

		ticks = append(ticks, newTicks...)
	}

	return ticks, nil
}

// getTickIndexesFromLogs returns all tick indexes from logs.
func (t *PoolTracker) getTickIndexesFromLogs(logs []ethtypes.Log) ([]int, error) {
	tickSet := make(map[int]struct{})
	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		switch event.Topics[0] {
		case poolV2ABI.Events["Mint"].ID:
			mint, err := poolFiltererV2.ParseMint(event)
			if err != nil {
				logger.WithFields(logger.Fields{"event": event, "error": err}).Error("failed to parse mint event")
				return nil, err
			}
			tickSet[int(mint.TickLower.Int64())] = struct{}{}
			tickSet[int(mint.TickUpper.Int64())] = struct{}{}

		case poolV3ABI.Events["Mint"].ID:
			mint, err := poolFiltererV3.ParseMint(event)
			if err != nil {
				logger.WithFields(logger.Fields{"event": event, "error": err}).Error("failed to parse mint event")
				return nil, err
			}
			tickSet[int(mint.TickLower.Int64())] = struct{}{}
			tickSet[int(mint.TickUpper.Int64())] = struct{}{}

		case poolV2ABI.Events["Burn"].ID:
			burn, err := poolFiltererV2.ParseBurn(event)
			if err != nil {
				logger.WithFields(logger.Fields{"event": event, "error": err}).Error("failed to parse burn event")
				return nil, err
			}
			tickSet[int(burn.TickLower.Int64())] = struct{}{}
			tickSet[int(burn.TickUpper.Int64())] = struct{}{}

		case poolV3ABI.Events["Burn"].ID:
			burn, err := poolFiltererV3.ParseBurn(event)
			if err != nil {
				logger.WithFields(logger.Fields{"event": event, "error": err}).Error("failed to parse burn event")
				return nil, err
			}
			tickSet[int(burn.TickLower.Int64())] = struct{}{}
			tickSet[int(burn.TickUpper.Int64())] = struct{}{}

		default:
			metrics.IncrUnprocessedEventTopic(DexTypeRamsesV2, event.Topics[0].Hex())
		}
	}

	return slices.Collect(maps.Keys(tickSet)), nil
}

func (t *PoolTracker) queryRPCTicksV2ByChunk(
	ctx context.Context, addr string, ticks []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	tickResponses := make([]TicksResp, len(ticks))

	ticksRequest := t.ethrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		ticksRequest.SetBlockNumber(&blockNumberBI)
	}

	for id, tick := range ticks {
		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    poolV2ABI,
			Target: addr,
			Method: methodTicks,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	l := logger.WithFields(logger.Fields{"address": addr})
	l.WithFields(logger.Fields{"len": len(ticksRequest.Calls), "ticks": ticks}).Debug("fetching ticks")

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksV2ByChunk(ctx, addr, ticks, 0)
		}

		logger.WithFields(logger.Fields{"error": err}).Error("failed to process aggregate to get ticks")
		return nil, err
	}

	result := make([]tickspkg.Tick, len(ticks))

	for id, tickResponse := range tickResponses {
		result[id] = tickspkg.Tick{
			TickIdx:        ticks[id],
			LiquidityGross: tickResponse.LiquidityGross,
			LiquidityNet:   tickResponse.LiquidityNet,
		}
	}

	return result, nil
}

func (t *PoolTracker) queryRPCTicksV3ByChunk(
	ctx context.Context, addr string, ticks []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	tickResponses := make([]TicksRespV3, len(ticks))

	ticksRequest := t.ethrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		ticksRequest.SetBlockNumber(&blockNumberBI)
	}

	for id, tick := range ticks {
		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    poolV3ABI,
			Target: addr,
			Method: methodTicks,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	l := logger.WithFields(logger.Fields{"address": addr})
	l.WithFields(logger.Fields{"len": len(ticksRequest.Calls), "ticks": ticks}).Debug("fetching ticks")

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksV3ByChunk(ctx, addr, ticks, 0)
		}

		logger.WithFields(logger.Fields{"error": err}).Error("failed to process aggregate to get ticks")
		return nil, err
	}

	result := make([]tickspkg.Tick, len(ticks))

	for id, tickResponse := range tickResponses {
		result[id] = tickspkg.Tick{
			TickIdx:        ticks[id],
			LiquidityGross: tickResponse.LiquidityGross,
			LiquidityNet:   tickResponse.LiquidityNet,
		}
	}

	return result, nil
}
