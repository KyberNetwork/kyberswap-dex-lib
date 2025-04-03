package uniswapv4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	bunniv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/custom-hook/bunni-v2"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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

func (t *PoolTracker) fetchRpcState(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)

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
		Params: []interface{}{eth.StringToBytes32(p.Address)},
	}, []interface{}{&result.Liquidity})

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: t.config.StateViewAddress,
		Method: "getSlot0",
		Params: []interface{}{eth.StringToBytes32(p.Address)},
	}, []interface{}{&result.Slot0})

	_, err := rpcRequests.Aggregate()
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
		rpcData, err = t.fetchRpcState(ctx, &p, 0)
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
		Liquidity:    rpcData.Liquidity,
		TickSpacing:  uint64(rpcData.TickSpacing),
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		Tick:         rpcData.Slot0.Tick,
		Ticks:        ticks,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)

	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to unmarshal static extra")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	switch {
	case hookAddress == bunniv2.HookAddress:
		if reserves, err := bunniv2.GetCustomReserves(ctx, p, t.ethrpcClient); err != nil {
			p.Reserves = reserves
		}
	default:
		// reserve0 = liquidity / sqrtPriceX96 * Q96
		reserve0 := new(big.Int).Mul(rpcData.Liquidity, Q96)
		reserve0.Div(reserve0, rpcData.Slot0.SqrtPriceX96)

		// reserve1 = liquidity * sqrtPriceX96 / Q96
		reserve1 := new(big.Int).Mul(rpcData.Liquidity, rpcData.Slot0.SqrtPriceX96)
		reserve1.Div(reserve1, Q96)

		p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}

	}

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
		return nil, ErrTooManyChangedTickes
	}

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

	stateViewTicks := make([]stateViewTick, changedTicksCount)
	for i, tickIdx := range changedTicks {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    stateViewABI,
			Target: t.config.StateViewAddress,
			Method: "getTickInfo",
			Params: []interface{}{eth.StringToBytes32(p.Address), big.NewInt(tickIdx)},
		}, []interface{}{&stateViewTicks[i]})
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
	liquidityGross := new(big.Int)
	liquidityGross, ok := liquidityGross.SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet := new(big.Int)
	liquidityNet, ok = liquidityNet.SetString(tickResp.LiquidityNet, 10)
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

func (t *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := t.fetchRpcState(ctx, &p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}
