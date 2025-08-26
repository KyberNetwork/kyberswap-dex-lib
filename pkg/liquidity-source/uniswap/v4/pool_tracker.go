package uniswapv4

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to unmarshal static extra")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	hookParam := &HookParam{Cfg: t.config, RpcClient: t.ethrpcClient, Pool: p}
	hook, _ := GetHook(hookAddress, hookParam)

	result := &FetchRPCResult{
		TickSpacing: staticExtra.TickSpacing,
	}
	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: t.config.StateViewAddress,
		Method: "getLiquidity",
		Params: []any{eth.StringToBytes32(p.Address)},
	}, []any{&result.Liquidity})

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: t.config.StateViewAddress,
		Method: "getSlot0",
		Params: []any{eth.StringToBytes32(p.Address)},
	}, []any{&result.Slot0})

	res, err := rpcRequests.Aggregate()
	if err != nil {
		return result, err
	}

	if result.Reserves, err = hook.GetReserves(ctx, hookParam); err != nil {
		return nil, err
	}
	if result.Reserves == nil { // default implementation is to estimate from liquidity and sqrtPriceX96
		var reserve0, reserve1 big.Int
		if result.Slot0.SqrtPriceX96.Sign() != 0 { // reserve0 = liquidity / sqrtPriceX96 * Q96
			reserve0.Mul(result.Liquidity, Q96)
			reserve0.Div(&reserve0, result.Slot0.SqrtPriceX96)
		}
		// reserve1 = liquidity * sqrtPriceX96 / Q96
		reserve1.Mul(result.Liquidity, result.Slot0.SqrtPriceX96)
		reserve1.Div(&reserve1, Q96)
		result.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	}

	hookParam.BlockNumber = res.BlockNumber
	result.HookExtra, err = hook.Track(ctx, hookParam)
	return result, err
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})
	l.Info("Start getting new state of univ4 pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		rpcData   *FetchRPCResult
		poolTicks []ticklens.TickResp
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
		if t.config.FetchTickFromStateView {
			poolTicks, err = t.getPoolTicksFromStateView(ctx, p, param)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

		poolTicks, err = t.getPoolTicks(ctx, p.Address)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for pool ticks")
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

	extraBytes, err := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			Liquidity:    rpcData.Liquidity,
			TickSpacing:  uint64(rpcData.TickSpacing),
			SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
			Tick:         rpcData.Slot0.Tick,
			Ticks:        ticks,
		},
		HookExtra: rpcData.HookExtra,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = rpcData.Reserves
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	p.IsInactive = t.IsInactive(&p, time.Now().Unix())

	l.Infof("Finish updating state of pool")
	return p, nil
}

// getPoolTicks
func (t *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]ticklens.TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []ticklens.TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []ticklens.TickResp `json:"ticks"`
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

type stateViewTick struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}

func (t *PoolTracker) getPoolTicksFromStateView(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) ([]ticklens.TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, errors.New("failed to unmarshal pool extra")
	}

	changedTicks := ticklens.GetChangedTicks(param.Logs)
	l.Infof("Fetch changed ticks %v", changedTicks)

	changedTicksCount := len(changedTicks)
	if changedTicksCount == 0 || changedTicksCount > maxChangedTicks {
		return nil, ErrTooManyChangedTicks
	}

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

	stateViewTicks := make([]stateViewTick, changedTicksCount)
	for i, tickIdx := range changedTicks {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    stateViewABI,
			Target: t.config.StateViewAddress,
			Method: "getTickInfo",
			Params: []any{eth.StringToBytes32(p.Address), big.NewInt(tickIdx)},
		}, []any{&stateViewTicks[i]})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	resTicks := make(map[int64]stateViewTick, len(resp.Request.Calls))
	for i, tick := range stateViewTicks {
		resTicks[changedTicks[i]] = tick
	}

	combined := make([]ticklens.TickResp, 0, len(changedTicks)+len(extra.Ticks))
	for _, t := range extra.Ticks {
		tIdx := int64(t.Index)
		if slices.Contains(changedTicks, tIdx) {
			tick := resTicks[tIdx]
			if tick.LiquidityNet == nil || tick.LiquidityNet.Sign() == 0 {
				// some changed ticks might be consumed entirely, delete them
				logger.Debugf("deleted tick %v %v", p.Address, t)
				continue
			}

			// changed, use new value
			combined = append(combined, ticklens.TickResp{
				TickIdx:        strconv.FormatInt(tIdx, 10),
				LiquidityGross: tick.LiquidityGross.String(),
				LiquidityNet:   tick.LiquidityNet.String(),
			})
		} else {
			// use old value
			combined = append(combined, ticklens.TickResp{
				TickIdx:        strconv.Itoa(t.Index),
				LiquidityGross: t.LiquidityGross.String(),
				LiquidityNet:   t.LiquidityNet.String(),
			})
		}
	}

	// Sort the ticks because function NewTickListDataProvider needs
	sort.SliceStable(combined, func(i, j int) bool {
		iTick, _ := strconv.Atoi(combined[i].TickIdx)
		jTick, _ := strconv.Atoi(combined[j].TickIdx)

		return iTick < jTick
	})

	return combined, nil
}

func transformTickRespToTick(tickResp ticklens.TickResp) (Tick, error) {
	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}

func (t *PoolTracker) IsInactive(p *entity.Pool, currentTimestamp int64) bool {
	if t.config.TrackInactivePools == nil || !t.config.TrackInactivePools.Enabled {
		return false
	}

	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.Errorf("failed to unmarshal static extra data")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	hookParam := &HookParam{Cfg: t.config, RpcClient: t.ethrpcClient, Pool: p}
	hook, _ := GetHook(hookAddress, hookParam)

	var inactiveTimeThresholdInSecond int64
	switch hook.GetExchange() {
	case valueobject.ExchangeUniswapV4Kem, valueobject.ExchangeUniswapV4FairFlow:
		return false
	case valueobject.ExchangeUniswapV4Zora:
		inactiveTimeThresholdInSecond = int64(t.config.TrackInactivePools.ZoraHookTimeThreshold.Seconds())
	default:
		inactiveTimeThresholdInSecond = int64(t.config.TrackInactivePools.TimeThreshold.Seconds())
	}

	return currentTimestamp-p.Timestamp > inactiveTimeThresholdInSecond
}

func (d *PoolTracker) GetInactivePools(_ context.Context, currentTimestamp int64,
	pools ...entity.Pool) ([]string, error) {
	if len(pools) == 0 {
		return nil, nil
	}

	inactivePools := lo.Filter(pools, func(p entity.Pool, _ int) bool {
		return d.IsInactive(&p, currentTimestamp)
	})

	return lo.Map(inactivePools, func(p entity.Pool, _ int) string { return p.Address }), nil
}
