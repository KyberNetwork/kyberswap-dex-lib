package uniswapv3

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/abis"
	ponsfun "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/forks/pons-fun"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// NewTracker never needs to know which fork it's tracking - unlike the factory/lister, it
// never stamps entity.Pool.Type (that's only ever set once, when a pool is first created),
// so the same constructor is registered for every DexType merged into this package.
var (
	_ = pooltrack.RegisterFactoryCEG(DexTypeUniswapV3, NewTracker)
	_ = pooltrack.RegisterFactoryCEG(DexTypePancakeV3, NewTracker)
	_ = pooltrack.RegisterFactoryCEG(DexTypeRamsesV2, NewTracker)
	_ = pooltrack.RegisterFactoryCEG(DexTypeSolidlyV3, NewTracker)
	_ = pooltrack.RegisterFactoryCEG(DexTypeSlipstream, NewTracker)
	_ = pooltrack.RegisterFactoryCEG(DexTypeNuriV2, NewTracker)
)

// int24TopicArgs/tickIndexTopics decode the indexed tickLower/tickUpper fields shared by
// every Mint/Burn shape merged into this package - see extractEventData. int24 is a signed
// type indexed as a full 32-byte two's-complement word, so this goes through go-ethereum's
// own topic decoder rather than a hand-rolled sign-extension.
var int24Type = lo.Must(ethabi.NewType("int24", "", nil))

var tickIndexTopicArgs = ethabi.Arguments{
	{Name: "tickLower", Type: int24Type, Indexed: true},
	{Name: "tickUpper", Type: int24Type, Indexed: true},
}

type tickIndexTopics struct {
	TickLower *big.Int
	TickUpper *big.Int
}

func extractTickIndexes(event ethtypes.Log) (lower, upper int, err error) {
	if len(event.Topics) < 4 {
		return 0, 0, ErrMalformedLog
	}

	var topics tickIndexTopics
	if err := ethabi.ParseTopics(&topics, tickIndexTopicArgs, event.Topics[2:4]); err != nil {
		return 0, 0, err
	}

	return int(topics.TickLower.Int64()), int(topics.TickUpper.Int64()), nil
}

type Tracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*Tracker, error) {
	initializedCfg, err := initializeConfig(config)
	if err != nil {
		return nil, err
	}

	return &Tracker{
		config:        initializedCfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func initializeConfig(cfg *Config) (*Config, error) {
	if cfg.PreGenesisPoolPath == "" {
		return cfg, nil
	}

	byteValue, ok := BytesByPath[cfg.PreGenesisPoolPath]
	if !ok {
		// Misconfiguration in the code, should check again
		return nil, errors.New("misconfigured PreGenesisPoolPath")
	}

	var pools []preGenesisPool
	if err := json.Unmarshal(byteValue, &pools); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to parse pools")
		return nil, err
	}

	logger.Infof("got %v pools from file: %s", len(pools), cfg.PreGenesisPoolPath)

	for _, p := range pools {
		cfg.preGenesisPoolIDs = append(cfg.preGenesisPoolIDs, p.ID)
	}

	return cfg, nil
}

func (t *Tracker) GetNewPoolState(ctx context.Context, p entity.Pool, param poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	if len(param.Logs) == 0 {
		return p, nil
	}

	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, param.Logs, l)
	if err != nil {
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, param.Logs, param.BlockHeaders)
}

func (t *Tracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// Reads through the uint256 shape - tolerant of both quoted and plain numbers - since some
	// already-persisted pools still carry the uint256-shaped Extra written before buildExtra
	// switched back to the big.Int Extra. Only Ticks[i].Index is used, unaffected either way.
	var extra ExtraTickU256
	if len(p.Extra) > 0 {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			return entity.Pool{}, err
		}
	}

	ticksToRefetch := make([]int, 0, len(extra.Ticks))
	for _, tick := range extra.Ticks {
		ticksToRefetch = append(ticksToRefetch, tick.Index)
	}

	if len(ticksToRefetch) == 0 {
		return p, nil
	}

	refetchedTicks, err := t.queryTicksFromRPC(ctx, p.Address, ticksToRefetch, p.BlockNumber)
	if err != nil {
		return entity.Pool{}, err
	}

	out := extraTickU256ToExtra(extra)
	out.Ticks = toTicks(refetchedTicks)

	extraBytes, err := json.Marshal(out)
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

// extraTickU256ToExtra carries the scalar fields of a tolerant uint256 read over to the big.Int
// Extra this package writes. Ticks is left for the caller to fill in.
func extraTickU256ToExtra(u ExtraTickU256) Extra {
	out := Extra{TickSpacing: u.TickSpacing, BuyRestrictedToken: u.BuyRestrictedToken}
	if u.Liquidity != nil {
		out.Liquidity = u.Liquidity.ToBig()
	}
	if u.SqrtPriceX96 != nil {
		out.SqrtPriceX96 = u.SqrtPriceX96.ToBig()
	}
	if u.Tick != nil {
		out.Tick = big.NewInt(int64(*u.Tick))
	}
	return out
}

func toTicks(ticks []tickspkg.Tick) []Tick {
	result := make([]Tick, 0, len(ticks))
	for _, tick := range ticks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		result = append(result, Tick{Index: tick.TickIdx, LiquidityGross: tick.LiquidityGross, LiquidityNet: tick.LiquidityNet})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })

	return result
}

func (t *Tracker) newTicksBasedPool(
	ctx context.Context,
	p entity.Pool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to transform entity pool to ticks based pool")
		return tickspkg.TicksBasedPool{}, err
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)

	ticksBasedPool.BlockNumber = blockNumber

	// Log ordering: [optional empty log] + [logs from reverted blocks] + [logs from new blocks]
	// If reorg happens, only extract affected tick ids from logs and fetch their state from RPC
	if eth.HasRevertedLog(logs) {
		return t.fetchTicksFromLogs(ctx, ticksBasedPool, logs, l)
	}

	return t.computeTicksFromLogs(ctx, ticksBasedPool, logs, l)
}

func (t *Tracker) fetchTicksFromLogs(
	ctx context.Context,
	tickBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	affectedTickIds, err := t.getAffectedTickIdsFromLogs(logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get affected tick IDs from logs")
		return tickBasedPool, err
	}

	if len(affectedTickIds) == 0 {
		return tickBasedPool, nil
	}

	l.WithFields(logger.Fields{
		"affectedTicks": affectedTickIds,
		"blockNumber":   tickBasedPool.BlockNumber,
	}).Info("fetching affected ticks from RPC for reverted blocks")

	affectedTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, affectedTickIds, tickBasedPool.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch affected ticks from RPC")
		return tickBasedPool, err
	}

	updateTicksMap(tickBasedPool.Ticks, affectedTicks)
	if tickBasedPool.HasValidTicks() {
		return tickBasedPool, err
	}

	l.WithFields(logger.Fields{
		"affectedTicks": affectedTickIds,
		"blockNumber":   tickBasedPool.BlockNumber,
	}).Warn("invalid pool ticks data after fetching ticks from logs")

	allTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, lo.Keys(tickBasedPool.Ticks), tickBasedPool.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch all ticks from RPC")
		return tickBasedPool, err
	}

	updateTicksMap(tickBasedPool.Ticks, allTicks)
	if !tickBasedPool.HasAllValidTicks() {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("invalid pool ticks data after fetching all ticks stored in pool")
		return tickBasedPool, err
	}

	return tickBasedPool, nil
}

func (t *Tracker) computeTicksFromLogs(
	ctx context.Context,
	tickBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})

	invalidTickSet := make(map[int]struct{})
	affectedTickSet := make(map[int]struct{})

	for _, event := range logs {
		if len(event.Topics) == 0 || valueobject.IsZeroAddress(event.Address) {
			continue
		}

		lower, upper, liquidityDelta, err := t.extractEventData(event)
		if err != nil {
			l.WithFields(logger.Fields{
				"blockNumber": event.BlockNumber,
				"logIndex":    event.Index,
				"error":       err,
			}).Error("failed to extract event data")
			continue
		}

		if liquidityDelta.Sign() == 0 {
			continue
		}

		affectedTickSet[lower] = struct{}{}
		affectedTickSet[upper] = struct{}{}

		if !t.applyLiquidityChange(tickBasedPool.Ticks, lower, liquidityDelta, true) {
			invalidTickSet[lower] = struct{}{}
		}
		if !t.applyLiquidityChange(tickBasedPool.Ticks, upper, liquidityDelta, false) {
			invalidTickSet[upper] = struct{}{}
		}
	}

	if len(affectedTickSet) == 0 {
		return tickBasedPool, nil
	}

	if !tickBasedPool.HasValidTicks() || len(invalidTickSet) > 0 {
		invalidTickIds := lo.Keys(invalidTickSet)
		affectedTickIds := lo.Keys(affectedTickSet)

		logFields := logger.Fields{
			"affectedTicks": affectedTickIds,
			"blockNumber":   tickBasedPool.BlockNumber,
		}
		if len(invalidTickIds) > 0 {
			logFields["invalidTicks"] = invalidTickIds
		}
		l.WithFields(logFields).Warn("tick state accumulated from logs invalid, fetching affected ticks from RPC")

		affectedTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, affectedTickIds, tickBasedPool.BlockNumber)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to refetch affected ticks from RPC")
			return tickBasedPool, err
		}

		updateTicksMap(tickBasedPool.Ticks, affectedTicks)

		if tickBasedPool.HasValidTicks() {
			return tickBasedPool, nil
		}

		l.WithFields(logger.Fields{
			"affectedTicks": affectedTickIds,
			"blockNumber":   tickBasedPool.BlockNumber,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		allTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, lo.Keys(tickBasedPool.Ticks), tickBasedPool.BlockNumber)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to refetch all ticks from RPC")
			return tickBasedPool, err
		}

		updateTicksMap(tickBasedPool.Ticks, allTicks)

		if !tickBasedPool.HasAllValidTicks() {
			l.WithFields(logger.Fields{
				"blockNumber": tickBasedPool.BlockNumber,
			}).Error("invalid pool ticks data after fetching all ticks stored in pool")

			return tickBasedPool, err
		}
	}

	return tickBasedPool, nil
}

func updateTicksMap(ticksMap map[int]tickspkg.Tick, newTicks []tickspkg.Tick) {
	for _, tick := range newTicks {
		ticksMap[tick.TickIdx] = tick
	}
}

func (t *Tracker) updateState(
	ctx context.Context,
	p entity.Pool,
	ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	// Always fetch scalar state (sqrtPrice/tick/liquidity/fee) at latest, never at this
	// batch's own block: a catch-up/backfill batch's block can be older than the
	// provider's pruning window, and there's no need for state "as of" that exact
	// block anyway since ticks are already applied incrementally from decoded logs.
	rpcState, err := t.FetchRPCData(ctx, &p, 0)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch state from RPC")
		return p, err
	}

	extra := buildExtra(rpcState, toTicksFromMap(ticksBasedPool.Ticks))
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.SwapFee, _ = rpcState.Fee.Float64()
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		rpcState.Reserve0.String(),
		rpcState.Reserve1.String(),
	}
	p.Timestamp = tickspkg.EstimateLastActivityTime(&p, logs, blockHeaders)
	p.BlockNumber = max(p.BlockNumber, lo.LastOrEmpty(logs).BlockNumber)

	return p, nil
}

func toTicksFromMap(ticks map[int]tickspkg.Tick) []Tick {
	return toTicks(lo.Values(ticks))
}

func buildExtra(rpcData *FetchRPCResult, ticks []Tick) Extra {
	var tick *big.Int
	if rpcData.Slot0.Unlocked {
		tick = rpcData.Slot0.Tick
	}

	var tickSpacing uint64
	if rpcData.TickSpacing != nil {
		tickSpacing = rpcData.TickSpacing.Uint64()
	}

	return Extra{
		Liquidity:          rpcData.Liquidity,
		SqrtPriceX96:       rpcData.Slot0.SqrtPriceX96,
		TickSpacing:        tickSpacing,
		Tick:               tick,
		Ticks:              ticks,
		BuyRestrictedToken: rpcData.BuyRestrictedToken,
	}
}

func (t *Tracker) getAffectedTickIdsFromLogs(logs []ethtypes.Log) ([]int, error) {
	affectedTickIds := make(map[int]struct{})

	for _, event := range logs {
		if len(event.Topics) == 0 || valueobject.IsZeroAddress(event.Address) {
			continue
		}

		lower, upper, liquidityDelta, err := t.extractEventData(event)
		if err != nil {
			return nil, err
		}

		if liquidityDelta.Sign() == 0 {
			continue
		}

		affectedTickIds[lower] = struct{}{}
		affectedTickIds[upper] = struct{}{}
	}

	return lo.Keys(affectedTickIds), nil
}

// extractEventData reads the tickLower/tickUpper/liquidityDelta touched by a Mint or Burn
// log. Burn has one shape across every merged fork; Mint has two (see mintEventID /
// mintWithIndexEventID in constant.go) that only differ in where `amount` sits within Data,
// since ramses-v2's static-fee pool inserts a non-indexed `index` field before it.
func (t *Tracker) extractEventData(event ethtypes.Log) (int, int, *big.Int, error) {
	if len(event.Topics) == 0 || valueobject.IsZeroAddress(event.Address) {
		return 0, 0, zeroBI, nil
	}

	switch event.Topics[0] {
	case burnEventID:
		lower, upper, err := extractTickIndexes(event)
		if err != nil {
			return 0, 0, nil, err
		}
		amount, err := readDataWord(event.Data, 0)
		if err != nil {
			return 0, 0, nil, err
		}
		return lower, upper, amount.Neg(amount), nil

	case mintEventID:
		lower, upper, err := extractTickIndexes(event)
		if err != nil {
			return 0, 0, nil, err
		}
		amount, err := readDataWord(event.Data, 1)
		if err != nil {
			return 0, 0, nil, err
		}
		return lower, upper, amount, nil

	case mintWithIndexEventID:
		lower, upper, err := extractTickIndexes(event)
		if err != nil {
			return 0, 0, nil, err
		}
		amount, err := readDataWord(event.Data, 2)
		if err != nil {
			return 0, 0, nil, err
		}
		return lower, upper, amount, nil

	default:
		metrics.IncrUnprocessedEventTopic(t.config.DexID, event.Topics[0].Hex())
		return 0, 0, zeroBI, nil
	}
}

// readDataWord reads the (wordIdx)-th unsigned 32-byte word from an event's non-indexed Data.
func readDataWord(data []byte, wordIdx int) (*big.Int, error) {
	start := wordIdx * 32
	if len(data) < start+32 {
		return nil, ErrMalformedLog
	}
	return new(big.Int).SetBytes(data[start : start+32]), nil
}

func (t *Tracker) applyLiquidityChange(
	ticks map[int]tickspkg.Tick,
	tickIdx int,
	liquidityDelta *big.Int,
	isLower bool,
) (isValid bool) {
	tick, exists := ticks[tickIdx]
	if !exists {
		tick = tickspkg.Tick{
			TickIdx:        tickIdx,
			LiquidityGross: big.NewInt(0),
			LiquidityNet:   big.NewInt(0),
		}
	}

	var newLiquidityGross big.Int
	newLiquidityGross.Add(tick.LiquidityGross, liquidityDelta)

	// exception: liquidityGross should never be negative
	if newLiquidityGross.Sign() < 0 {
		return false
	}

	tick.LiquidityGross.Set(&newLiquidityGross)

	if isLower {
		tick.LiquidityNet.Add(tick.LiquidityNet, liquidityDelta)
	} else {
		tick.LiquidityNet.Sub(tick.LiquidityNet, liquidityDelta)
	}

	ticks[tickIdx] = tick

	return true
}

// queryTicksFromRPC returns ticks data of `tickIndexes` in pool `address` at `blockNumber`.
// If `blockNumber` == 0, it returns the latest ticks data.
func (t *Tracker) queryTicksFromRPC(
	ctx context.Context,
	address string,
	tickIndexes []int,
	blockNumber uint64,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		return t.queryRPCTicksByChunk(ctx, address, tickIndexes, blockNumber)
	}

	var result []tickspkg.Tick
	for i := 0; i < len(tickIndexes); i += tickChunkSize {
		end := min(i+tickChunkSize, len(tickIndexes))
		ticks, err := t.queryRPCTicksByChunk(ctx, address, tickIndexes[i:end], blockNumber)
		if err != nil {
			return nil, err
		}

		result = append(result, ticks...)
	}

	return result, nil
}

// queryRPCTicksByChunk returns liquidityGross/liquidityNet for a chunk of ticks. Every merged
// fork's ticks() returns liquidityGross/liquidityNet in the same first two positions
// regardless of how many (or which) fields follow, so one minimal 2-field ABI decodes all of
// them - unlike slot0()'s fee field, nothing else we read is position-sensitive here.
func (t *Tracker) queryRPCTicksByChunk(
	ctx context.Context,
	address string,
	ticks []int,
	blockNumber uint64,
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
			ABI:    abis.UniswapV3PoolABI,
			Target: address,
			Method: methodTicks,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksByChunk(ctx, address, ticks, 0)
		}

		return nil, fmt.Errorf("failed to process aggregate to get ticks: %w", err)
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

func (t *Tracker) BootstrapPoolState(ctx context.Context, p entity.Pool, param poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
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
		// Ad-hoc logic to handle edge case on Optimism
		// Link to issue: https://www.notion.so/kybernetwork/Aggregator-1-20-defect-1caec6062f9d4da0918fc3443e6e1963#0810d1462cc14f0a9465f935c9e641fe
		// TLDR: Optimism has some pre-genesis Uniswap V3 pool. Subgraph does not have data for these pools
		// So we have to fetch ticks data from the TickLens smart contract (which is slower).
		if t.config.AlwaysUseTickLens || lo.Contains(t.config.preGenesisPoolIDs, p.Address) {
			poolTicks, err = ticklens.GetPoolTicksFromSC(ctx, t.ethrpcClient, t.config.TickLensAddress, p, nil)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
		} else {
			// If pool is not pre-genesis, fetch from subgraph
			poolTicks, err = t.getPoolTicks(ctx, p.Address)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to query subgraph for pool ticks")
			}
		}

		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	var ticks []Tick
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(buildExtra(rpcData, ticks))
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.SwapFee, _ = rpcData.Fee.Float64()
	p.Extra = string(extraBytes)
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(param.Logs).BlockTimestamp))
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = max(p.BlockNumber, lo.LastOrEmpty(param.Logs).BlockNumber)

	l.Infof("Finish updating state of pool")

	return p, nil
}

// FetchRPCData fetches liquidity/slot0/tickSpacing/fee/reserves generically for every fork
// merged into this package - see Slot0 and FetchRPCResult in type.go for how slot0's shape
// and the fee source are resolved without needing to know which fork this pool belongs to.
func (t *Tracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var (
		liquidity   *big.Int
		tickSpacing *big.Int
		fee         *big.Int
		currentFee  *big.Int
		reserve0    = zeroBI
		reserve1    = zeroBI

		slot0Std    slot0RawStandard
		slot0Katana slot0RawKatana
		slot0Slip   slot0RawSlipstream
		slot0Solid  slot0RawSolidly
	)

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
	}, []any{&liquidity})

	// Try katana (8-word) first, then standard (7-word), then slipstream (6-word), then solidly (4-word) shapes, longest
	// first - go-ethereum ABI decoder only errors on insufficient data, not unconsumed trailing bytes.
	// A shorter ABI silently misdecodes longer returndata by putting wrong bytes into its last fields.
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:       abis.UniswapV3PoolABI,
		UnpackABI: []ethabi.ABI{abis.Slot0KatanaABI, abis.UniswapV3PoolABI, abis.Slot0SlipstreamABI, abis.Slot0SolidlyABI},
		Target:    p.Address,
		Method:    methodGetSlot0,
	}, []any{&slot0Katana, &slot0Std, &slot0Slip, &slot0Solid})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodTickSpacing,
	}, []any{&tickSpacing})

	// fee()/currentFee() are both attempted; a pool only ever answers the subset it
	// actually implements (the others simply revert). See FetchRPCResult.Fee in type.go
	// for the resolution priority and why it must not just be "whichever succeeds first".
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodFee,
	}, []any{&fee})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodCurrentFee,
	}, []any{&currentFee})

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

	var ponsGuard *ponsfun.Guard
	if fork, ok := t.config.ForksConfig[valueobject.ExchangePonsFun]; ok && len(p.Tokens) == 2 {
		ponsGuard = ponsfun.NewGuard(t.config.ChainID, fork.Multicall3, p.Tokens[0].Address, p.Tokens[1].Address)
		ponsGuard.AddCalls(rpcRequest)
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return nil, err
	}

	slot0, err := resolveSlot0(slot0Katana, slot0Std, slot0Slip, slot0Solid)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to decode slot0 with any known shape")
		return nil, err
	}

	return &FetchRPCResult{
		Liquidity:          liquidity,
		Slot0:              slot0,
		TickSpacing:        tickSpacing,
		Fee:                resolveFee(fee, currentFee, slot0.Fee),
		Reserve0:           reserve0,
		Reserve1:           reserve1,
		BuyRestrictedToken: ponsGuard.BuyRestrictedToken(),
	}, nil
}

func resolveSlot0(katana slot0RawKatana, std slot0RawStandard, slip slot0RawSlipstream, solid slot0RawSolidly) (Slot0, error) {
	switch {
	case katana.SqrtPriceX96 != nil:
		return Slot0{SqrtPriceX96: katana.SqrtPriceX96, Tick: katana.Tick, Unlocked: katana.Unlocked}, nil
	case std.SqrtPriceX96 != nil:
		return Slot0{SqrtPriceX96: std.SqrtPriceX96, Tick: std.Tick, Unlocked: std.Unlocked}, nil
	case slip.SqrtPriceX96 != nil:
		return Slot0{SqrtPriceX96: slip.SqrtPriceX96, Tick: slip.Tick, Unlocked: slip.Unlocked}, nil
	case solid.SqrtPriceX96 != nil:
		return Slot0{SqrtPriceX96: solid.SqrtPriceX96, Tick: solid.Tick, Unlocked: solid.Unlocked, Fee: solid.Fee}, nil
	default:
		return Slot0{}, errors.New("slot0 did not decode with any known shape")
	}
}

// resolveFee picks the pool's real fee. currentFee() must win over fee() when both succeed:
// ramses-v2's dynamic pool and nuri-v2 implement both, but their fee() is a stale value fixed
// at deploy time, not the live dynamic fee - only currentFee() is current. Falls back to
// slot0's embedded fee (solidly-v3, which has neither method) if both calls reverted.
func resolveFee(fee, currentFee, slot0Fee *big.Int) *big.Int {
	switch {
	case currentFee != nil:
		return currentFee
	case fee != nil:
		return fee
	case slot0Fee != nil:
		return slot0Fee
	default:
		return zeroBI
	}
}

func (t *Tracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []TickResp                `json:"ticks"`
			Meta  *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Ticks == nil {
					l.WithFields(logger.Fields{
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Error("failed to query subgraph")

					return nil, err
				}
			} else {
				l.WithFields(logger.Fields{
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Error("failed to query subgraph")

				return nil, err
			}
		}

		resp.Meta.CheckIsLagging(t.config.DexID, poolAddress)

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
