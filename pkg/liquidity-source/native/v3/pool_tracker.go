package v3

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexType, NewPoolTracker)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var (
		slot0                  Slot0
		liquidity, tickSpacing *big.Int

		reserves            = [2]*big.Int{common.Big0, common.Big0}
		underlyingTokens    = make([]common.Address, len(p.Tokens))
		isUnderlyingScanned = IsUnderlyingScanned(ctx)

		vaultRPCs = [2]VaultRPC{}
	)

	rpcRequest := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetLiquidity,
		}, []any{&liquidity}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetSlot0,
		}, []any{&slot0}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodTickSpacing,
		}, []any{&tickSpacing})

	start := 0
	if len(p.Tokens) == 4 {
		start = 2
	}

	for i := start; i < len(p.Tokens); i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[i].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserves[i-start]})

		if !isUnderlyingScanned {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodUnderlying,
				Params: nil,
			}, []any{&underlyingTokens[i-start]})
		}

		rpcRequest.
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodMinDeposit,
			}, []any{&vaultRPCs[i-start].MinDeposit}).
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodDepositPaused,
			}, []any{&vaultRPCs[i-start].DepositPaused}).
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodRedeemPaused,
			}, []any{&vaultRPCs[i-start].RedeemPaused}).
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodExchangeRate,
			}, []any{&vaultRPCs[i-start].ExchangeRate}).
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodMinRedeemInterval,
			}, []any{&vaultRPCs[i-start].MinRedeemInterval}).
			AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodRedeemCoolDownExempt,
				Params: []any{common.HexToAddress(t.config.ExecutorAddress)},
			}, []any{&vaultRPCs[i-start].RedeemCoolDownExempt})
	}

	res, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process TryBlockAndAggregate")
		return nil, err
	}

	var vaults = [2]Vault{}
	for i, v := range vaultRPCs {
		vaults[i].DepositPaused = v.DepositPaused
		vaults[i].RedeemPaused = v.RedeemPaused
		vaults[i].MinDeposit = uint256.MustFromBig(v.MinDeposit)
		vaults[i].ExchangeRate = uint256.MustFromBig(v.ExchangeRate)
		vaults[i].MinRedeemInterval = uint256.MustFromBig(v.MinRedeemInterval)
		vaults[i].RedeemCoolDownExempt = v.RedeemCoolDownExempt
	}

	return &FetchRPCResult{
		Liquidity:        liquidity,
		Slot0:            slot0,
		Reserves:         reserves,
		UnderlyingTokens: underlyingTokens,
		BlockNumber:      res.BlockNumber.Uint64(),
		Vaults:           vaults,
	}, nil
}

func (t *PoolTracker) fetchPoolTicks(ctx context.Context, p entity.Pool, _ sourcePool.GetNewPoolStateParams) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	poolTicks, err := t.getPoolTicks(ctx, p.Address)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to query subgraph for pool ticks")
		return nil, err
	}
	return poolTicks, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pool")

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = t.FetchRPCData(ctx, &p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")
		}
		return err
	})
	g.Go(func(context.Context) error {
		var err error
		poolTicks, err = t.fetchPoolTicks(ctx, p, param)
		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	ticks := lo.Map(poolTicks, func(tickResp TickResp, _ int) Tick {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			return Tick{}
		}
		return tick
	})

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcData.Liquidity,
		Unlocked:     rpcData.Slot0.Unlocked,
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		Tick:         rpcData.Slot0.Tick,
		Ticks:        ticks,
		Vaults:       rpcData.Vaults,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	if rpcData.Slot0.Unlocked {
		p.Reserves = lo.Map(p.Tokens, func(_ *entity.PoolToken, i int) string {
			if i < 2 {
				return rpcData.Reserves[i].String()
			}
			return "1"
		})
	} else {
		p.Reserves = lo.Map(p.Tokens, func(_ *entity.PoolToken, _ int) string {
			return "0"
		})
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = rpcData.BlockNumber

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var response struct {
			Ticks []TickResp `json:"ticks"`
		}

		if err := t.graphqlClient.Run(ctx, req, &response); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError && len(response.Ticks) > 0 {
				ticks = append(ticks, response.Ticks...)
				break
			}

			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph")
			return nil, err
		}

		if len(response.Ticks) == 0 {
			break
		}

		ticks = append(ticks, response.Ticks...)
		lastTickIdx = response.Ticks[len(response.Ticks)-1].TickIdx

		if len(response.Ticks) < graphFirstLimit {
			break
		}
	}

	return ticks, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

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
			return entity.Pool{}, err
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
		return entity.Pool{}, err
	}

	// convert back to native-v3 ticks
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
		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
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
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to transform entity pool to ticks based pool")
		return ticksBasedPool, err
	}

	ticks, err := t.fetchTicksFromLogs(ctx, p.Address, logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to FetchTicksFromLogs")
		return ticksBasedPool, err
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)
	ticksBasedPool.BlockNumber = blockNumber

	if len(ticks) == 0 {
		return ticksBasedPool, nil
	}

	if err := tickspkg.ValidatePoolTicks(ticksBasedPool, ticks); err != nil {
		l.WithFields(logger.Fields{
			"numTicks": len(ticks),
			"error":    err,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		l.WithFields(logger.Fields{
			"numTicks": len(ticksBasedPool.Ticks),
		}).Info("fetch all ticks for pool")

		ticks, err = t.fetchAllTicksForPool(ctx, ticksBasedPool, ticks)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch all ticks")

			return ticksBasedPool, err
		}

		if err := tickspkg.ValidateAllPoolTicks(ticksBasedPool, ticks); err != nil {
			l.WithFields(logger.Fields{
				"numTicks": len(ticks),
				"error":    err,
			}).Warnf("invalid pool ticks data after fetching all ticks stored in pool")
		}
	}

	for _, tick := range ticks {
		ticksBasedPool.Ticks[tick.TickIdx] = tick
	}

	return ticksBasedPool, nil
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

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	blockNumber := ticksBasedPool.BlockNumber

	var (
		rpcState    *FetchRPCResult
		staticExtra StaticExtra
	)

	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to unmarshal static extra data")
		return p, err
	}

	newCtx := NewContextWithUnderlyingScanned(ctx, staticExtra.NeedScanUnderlying)
	rpcState, err = t.FetchRPCData(newCtx, &p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, &p, 0)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to fetch latest state from RPC")
				return p, err
			}
		} else {
			l.WithFields(logger.Fields{
				"error":       err,
				"blockNumber": blockNumber,
			}).Error("failed to fetch state from RPC")
			return p, err
		}
	}

	// Only process underlying tokens if we haven't checked before
	if len(p.Tokens) == 2 && !staticExtra.NeedScanUnderlying {
		underlyingTokens := make([]*entity.PoolToken, 0, len(rpcState.UnderlyingTokens))

		for i := range rpcState.UnderlyingTokens {
			if rpcState.UnderlyingTokens[i] == valueobject.AddrZero {
				continue
			}

			underlyingTokens = append(underlyingTokens, &entity.PoolToken{
				Address:   hexutil.Encode(rpcState.UnderlyingTokens[i][:]),
				Decimals:  p.Tokens[i].Decimals,
				Swappable: true,
			})
		}

		staticExtra.NeedScanUnderlying = true

		if len(underlyingTokens) > 0 {
			p.Tokens = append(underlyingTokens, p.Tokens...)
		}
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal static extra data")
		return p, err
	}

	entityPoolTicks := make([]Tick, 0, len(ticksBasedPool.Ticks))
	for _, tick := range ticksBasedPool.Ticks {
		// skip uninitialized ticks
		// TODO: subgraph can fail to handle re-org blocks and leads to some ticks were uninitialized but it is still initialized on on-chain (except we re-fetch all ticks from on-chain when running bootstrap). We might remove this skip in testing phase if it causes the issue.
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
		Unlocked:     rpcState.Slot0.Unlocked,
		SqrtPriceX96: rpcState.Slot0.SqrtPriceX96,
		Tick:         rpcState.Slot0.Tick,
		Ticks:        entityPoolTicks,
		Vaults:       rpcState.Vaults,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	if rpcState.Slot0.Unlocked {
		p.Reserves = lo.Map(p.Tokens, func(_ *entity.PoolToken, i int) string {
			if i < 2 {
				return rpcState.Reserves[i].String()
			}
			return "1"
		})
	} else {
		p.Reserves = lo.Map(p.Tokens, func(_ *entity.PoolToken, _ int) string {
			return "0"
		})
	}

	p.StaticExtra = string(staticExtraBytes)
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = rpcState.BlockNumber

	return p, nil
}

func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context,
	address string,
	logs []ethtypes.Log,
) ([]tickspkg.Tick, error) {
	l := logger.WithFields(logger.Fields{
		"address": address,
	})

	if len(logs) == 0 {
		return nil, nil
	}

	tickIndexes, err := t.getTickIndexesFromLogs(logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getTickIndexesFromEvents")
		return nil, err
	}

	if len(tickIndexes) == 0 {
		return nil, nil
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)

	return t.queryRPCTicksByIndexes(ctx, address, tickIndexes, blockNumber)
}

// getTickIndexesFromLogs returns all tick indexes from logs.
func (t *PoolTracker) getTickIndexesFromLogs(logs []ethtypes.Log) ([]int, error) {
	tickSet := make(map[int]struct{})
	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		switch event.Topics[0] {
		case poolABI.Events["Mint"].ID:
			mint, err := poolFilterer.ParseMint(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse mint event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   mint,
			}).Debug("decode mint event")

			tickSet[int(mint.TickLower.Int64())] = struct{}{}
			tickSet[int(mint.TickUpper.Int64())] = struct{}{}

		case poolABI.Events["Burn"].ID:
			burn, err := poolFilterer.ParseBurn(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse burn event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   burn,
			}).Debug("decode burn event")

			tickSet[int(burn.TickLower.Int64())] = struct{}{}
			tickSet[int(burn.TickUpper.Int64())] = struct{}{}

		default:
			metrics.IncrUnprocessedEventTopic(DexType, event.Topics[0].Hex())
		}
	}

	ticks := make([]int, 0, len(tickSet))
	for tick := range tickSet {
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

// queryRPCTicksByIndexes returns ticks data of `tickIndexes` in pool `address` at `blockNumber`.
// If `blockNumber` == 0, it returns the latest ticks data.
func (t *PoolTracker) queryRPCTicksByIndexes(
	ctx context.Context, address string, tickIndexes []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		return t.queryRPCTicksByChunk(ctx, address, tickIndexes, blockNumber)
	}

	totalTicks := len(tickIndexes)
	ticks := make([]tickspkg.Tick, 0, totalTicks)
	for i := 0; i < totalTicks; i += tickChunkSize {
		toIdx := i + tickChunkSize
		if toIdx > totalTicks {
			toIdx = totalTicks
		}

		newTicks, err := t.queryRPCTicksByChunk(ctx, address, tickIndexes[i:toIdx], blockNumber)
		if err != nil {
			return nil, err
		}

		ticks = append(ticks, newTicks...)
	}

	return ticks, nil
}

// queryRPCTicksByChunk returns native-v3 Ticks data.
func (t *PoolTracker) queryRPCTicksByChunk(
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
			ABI:    poolABI,
			Target: addr,
			Method: poolMethodTicks,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	l := logger.WithFields(logger.Fields{
		"address": addr,
	})

	l.WithFields(logger.Fields{
		"len":   len(ticksRequest.Calls),
		"ticks": ticks,
	}).Debug("fetching ticks")

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksByChunk(ctx, addr, ticks, 0)
		}

		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process aggregate to get ticks")
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
