package v3

import (
	"context"
	"fmt"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
	lensGuard     *ephemeral.Guard
	// readTicks reads ticks by index at a block, queryRPCTicksByIndexes but for tests.
	readTicks func(ctx context.Context, address string, tickIndexes []int, blockNumber uint64) ([]tickspkg.Tick,
		error)
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	t := &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
		lensGuard:     ephemeral.NewLensGuard(cfg.LensGuard, cfg.LensBatch),
	}
	t.readTicks = t.queryRPCTicksByIndexes
	return t
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

func (t *PoolTracker) BootstrapPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pool")

	if t.config.UseLens {
		lensPool, _, err := t.bootstrapFromLens(ctx, p, param, nil, l)
		if err == nil {
			return lensPool, nil
		}
		// A lens read can fail where the old path does not, as one paging across blocks does on a
		// node that ignores a pinned block, HyperEVM's: the pool is then bootstrapped without it
		// rather than left unsaved.
		l.WithFields(logger.Fields{"error": err}).Warn("failed to bootstrap the pool through the lens, bootstrapping it without")
	}

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
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(param.Logs).BlockTimestamp))
	p.BlockNumber = max(p.BlockNumber, lo.LastOrEmpty(param.Logs).BlockNumber)

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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, param sourcePool.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	if t.config.UseLens && !tickspkg.HasState(p) {
		return tickspkg.StateFromLens(ctx, t.lensGuard, p, param,
			func(ctx context.Context, block *big.Int) (entity.Pool, uint64, error) {
				return t.bootstrapFromLens(ctx, p, param, block, l)
			},
			l), nil
	}

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, param.Logs)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, param.Logs, param.BlockHeaders)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// The lens read the ticks at the block it read the state, which is what refetching is for.
	if t.config.UseLens {
		return p, nil
	}

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

	blockNumber := tickspkg.ReadBlock(logs, p.BlockNumber)
	ticksBasedPool.BlockNumber = blockNumber

	ticks, err := t.fetchTicksFromLogs(ctx, p.Address, logs, blockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to FetchTicksFromLogs")
		return ticksBasedPool, err
	}

	if len(ticks) == 0 {
		return ticksBasedPool, nil
	}

	if err := tickspkg.ValidatePoolTicks(ticksBasedPool, ticks); err != nil {
		l.WithFields(logger.Fields{
			"numTicks": len(ticks),
			"error":    err,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		if t.ticksFromLens(ctx, &ticksBasedPool, blockNumber) {
			return ticksBasedPool, nil
		}

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

// ticksFromLens replaces the pool's ticks with every initialized one the lens reads at block, once
// those from logs no longer add up: refetching the stored indexes cannot find a tick the stored set
// never had. The read goes through the guard, so it never makes the update wait. It reports
// whether the ticks were replaced; when not, the stored ones are refetched as without the lens.
func (t *PoolTracker) ticksFromLens(ctx context.Context, p *tickspkg.TicksBasedPool, block uint64) bool {
	if !t.config.UseLens {
		return false
	}
	ran, err := t.lensGuard.Try(ctx, p.Address, func(ctx context.Context) error {
		snap, err := t.snapshot(ctx, p.Address, new(big.Int).SetUint64(block))
		if err != nil {
			return err
		}
		p.Ticks = make(map[int]tickspkg.Tick, len(snap.ticks))
		for _, tick := range snap.ticks {
			p.Ticks[tick.Index] = tickspkg.Tick{
				TickIdx:        tick.Index,
				LiquidityGross: tick.LiquidityGross,
				LiquidityNet:   tick.LiquidityNet,
			}
		}
		return nil
	})
	if ran && err == nil {
		return true
	}
	logger.WithFields(logger.Fields{"address": p.Address, "ran": ran, "error": err}).
		Warn("ticks not read through the lens; refetching the stored ones")
	return false
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

	ticksFromPool, err := t.readTicks(ctx, pool.Address, tickIdsFromPool, pool.BlockNumber)
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

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log, blockHeaders map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

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

	// Always fetch scalar state at latest, never at this batch's own block: see the
	// comment on the equivalent fetch in uniswap/v3/pool_tracker.go's updateState.
	newCtx := NewContextWithUnderlyingScanned(ctx, staticExtra.NeedScanUnderlying)
	rpcState, err = t.FetchRPCData(newCtx, &p, 0)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch state from RPC")
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

	p, err = withState(p, rpcState, staticExtra, entityPoolTicks)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal pool state")
		return p, err
	}

	p.Timestamp = tickspkg.EstimateLastActivityTime(&p, logs, blockHeaders)
	p.BlockNumber = max(p.BlockNumber, lo.LastOrEmpty(logs).BlockNumber)

	return p, nil
}

// withState writes rpcState and ticks into p. A pool created from its factory log holds its two LP
// tokens only; the first state it gets puts their underlying tokens in front, once.
func withState(p entity.Pool, rpcState *FetchRPCResult, staticExtra StaticExtra, ticks []Tick) (entity.Pool,
	error) {
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
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcState.Liquidity,
		Unlocked:     rpcState.Slot0.Unlocked,
		SqrtPriceX96: rpcState.Slot0.SqrtPriceX96,
		Tick:         rpcState.Slot0.Tick,
		Ticks:        ticks,
		Vaults:       rpcState.Vaults,
	})
	if err != nil {
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
	return p, nil
}

// bootstrapFromLens reads the pool's state and ticks through the lens at block (nil for the
// latest), then its vaults and reserves, which the lens does not return, at the same block. It
// returns the pool with the block the lens read it at.
func (t *PoolTracker) bootstrapFromLens(ctx context.Context, p entity.Pool, param sourcePool.GetNewPoolStateParams,
	block *big.Int, l logger.Logger) (entity.Pool, uint64, error) {
	snap, err := t.snapshot(ctx, p.Address, block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to read pool through the lens")
		return entity.Pool{}, 0, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, 0, fmt.Errorf("static extra of %s: %w", p.Address, err)
	}

	rpcData, err := t.FetchRPCData(NewContextWithUnderlyingScanned(ctx, staticExtra.NeedScanUnderlying), &p,
		snap.block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch data from RPC")
		return entity.Pool{}, 0, err
	}
	if rpcData.Liquidity == nil || rpcData.Slot0.Tick == nil || rpcData.Liquidity.Cmp(snap.liquidity) != 0 ||
		rpcData.Slot0.Tick.Int64() != int64(snap.tick) {
		return entity.Pool{}, 0, fmt.Errorf("pool %s at block %d: RPC and lens disagree on liquidity or tick",
			p.Address, snap.block)
	}

	if p, err = withState(p, rpcData, staticExtra, snap.ticks); err != nil {
		return entity.Pool{}, 0, err
	}
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(param.Logs).BlockTimestamp))
	p.BlockNumber = max(p.BlockNumber, snap.block, lo.LastOrEmpty(param.Logs).BlockNumber)

	l.Infof("Finish updating state of pool")
	return p, snap.block, nil
}

var _ sourcePool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)

// NextUpdateAt asks for a pool still without state, its lens read declined by the guard, to be
// updated again as soon as the guard would take it: it may see no log for a long time, and only an
// update gives it its state. Every other pool needs no update without an event.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	if t.config.UseLens && !tickspkg.HasState(p) {
		return t.lensGuard.ReadyAt(p.Address, time.Now()).Unix()
	}
	return 0
}

var _ sourcePool.IPoolTrackerWithReset = (*PoolTracker)(nil)

// ResetPoolState drops the pool's state for its next update to read it whole through the lens, at
// a block no lower than block. Without the lens nothing can read it whole, and the pool is kept.
func (t *PoolTracker) ResetPoolState(p entity.Pool, block uint64) (entity.Pool, bool) {
	if !t.config.UseLens {
		return p, false
	}
	return tickspkg.WithoutState(p, block), true
}

// fetchTicksFromLogs reads the ticks logs touch at blockNumber (see tickspkg.ReadBlock).
func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context,
	address string,
	logs []ethtypes.Log,
	blockNumber uint64,
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

	return t.readTicks(ctx, address, tickIndexes, blockNumber)
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
		toIdx := min(i+tickChunkSize, totalTicks)

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
