package cl

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	result := &FetchRPCResult{
		TickSpacing: staticExtra.TickSpacing,
	}
	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.CLPoolManagerABI,
		Target: t.config.CLPoolManagerAddress,
		Method: shared.CLPoolManagerMethodGetLiquidity,
		Params: []any{common.HexToHash(p.Address)},
	}, []any{&result.Liquidity})

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.CLPoolManagerABI,
		Target: t.config.CLPoolManagerAddress,
		Method: shared.CLPoolManagerMethodGetSlot0,
		Params: []any{common.HexToHash(p.Address)},
	}, []any{&result.Slot0})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		return nil, err
	}

	protocolFee, lpFee := uint64(result.Slot0.ProtocolFee&_MASK12), uint64(result.Slot0.LpFee)
	if shared.IsDynamicFee(staticExtra.Fee) {
		lpFee = uint64(t.GetDynamicFee(ctx, staticExtra.HooksAddress, uint32(lpFee)))
	}

	// https://github.com/pancakeswap/infinity-core/blob/6d0b5ee/src/libraries/ProtocolFeeLibrary.sol#L52
	result.SwapFee = uint32(protocolFee + lpFee - (protocolFee * lpFee / 1_000_000))

	return result, nil
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

	l.Info("Start getting new state of pancake-infinity-cl pool")

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
		if t.config.FetchTickFromRPC {
			poolTicks, err = t.getPoolTicksFromRPC(ctx, p, param)
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

	var ticks = make([]Tick, 0, len(poolTicks))
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

	extra := Extra{
		Liquidity:    rpcData.Liquidity,
		TickSpacing:  rpcData.TickSpacing,
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		Tick:         rpcData.Slot0.Tick,
		Ticks:        ticks,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.SwapFee = float64(rpcData.SwapFee)

	p.Extra = string(extraBytes)

	var reserve0, reserve1 big.Int
	if rpcData.Slot0.SqrtPriceX96.Sign() != 0 {
		// reserve0 = liquidity / sqrtPriceX96 * Q96
		reserve0.Mul(rpcData.Liquidity, Q96)
		reserve0.Div(&reserve0, rpcData.Slot0.SqrtPriceX96)
	}

	// reserve1 = liquidity * sqrtPriceX96 / Q96
	reserve1.Mul(rpcData.Liquidity, rpcData.Slot0.SqrtPriceX96)
	reserve1.Div(&reserve1, Q96)

	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	p.BlockNumber = blockNumber

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

func (t *PoolTracker) GetDynamicFee(ctx context.Context, hookAddress common.Address, lpFee uint32) uint32 {
	hook, _ := GetHook(hookAddress)
	return hook.GetDynamicFee(ctx, t.ethrpcClient, t.config.CLPoolManagerAddress, hookAddress, lpFee)
}

type rpcTick struct {
	Data struct {
		LiquidityGross        *big.Int
		LiquidityNet          *big.Int
		FeeGrowthOutside0X128 *big.Int
		FeeGrowthOutside1X128 *big.Int
	}
}

func (t *PoolTracker) getPoolTicksFromRPC(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) ([]ticklens.TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			return nil, err
		}
	}

	changedTicks := ticklens.GetChangedTicks(param.Logs)
	l.Infof("Fetch changed ticks %v", changedTicks)

	changedTicksCount := len(changedTicks)
	if changedTicksCount == 0 || changedTicksCount > maxChangedTicks {
		return nil, ErrTooManyChangedTickes
	}

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

	rpcTicks := make([]rpcTick, changedTicksCount)
	for i, tickIdx := range changedTicks {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    shared.CLPoolManagerABI,
			Target: t.config.CLPoolManagerAddress,
			Method: shared.CLPoolManagerMethodGetPoolTickInfo,
			Params: []any{common.HexToHash(p.Address), big.NewInt(tickIdx)},
		}, []any{&rpcTicks[i]})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	resTicks := make(map[int64]rpcTick, len(resp.Request.Calls))
	for i, tick := range rpcTicks {
		resTicks[changedTicks[i]] = tick
	}

	combined := make([]ticklens.TickResp, 0, len(changedTicks)+len(extra.Ticks))
	for _, t := range extra.Ticks {
		tIdx := int64(t.Index)
		if slices.Contains(changedTicks, tIdx) {
			tick := resTicks[tIdx]
			if tick.Data.LiquidityNet == nil || tick.Data.LiquidityNet.Sign() == 0 {
				// some changed ticks might be consumed entirely, delete them
				logger.Debugf("deleted tick %v %v", p.Address, t)
				continue
			}

			// changed, use new value
			combined = append(combined, ticklens.TickResp{
				TickIdx:        strconv.FormatInt(tIdx, 10),
				LiquidityGross: tick.Data.LiquidityGross.String(),
				LiquidityNet:   tick.Data.LiquidityNet.String(),
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
