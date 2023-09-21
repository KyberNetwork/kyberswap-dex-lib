package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Elastic] Start getting new state of pool: %v", p.Address)

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p)
		if err != nil {
			logger.Errorf("failed to fetch data from RPC for pool: %v, err: %v", p.Address, err)
		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		poolTicks, err = d.getPoolTicks(ctx, p.Address)
		if err != nil {
			logger.Errorf("failed to query subgraph for pool ticks, pool: %v, err: %v", p.Address, err)
		}

		return err
	})

	if err := g.Wait(); err != nil {
		logger.Errorf("failed to fetch pool state, pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	var ticks []Tick
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			logger.Errorf("failed to transform tickResp to tick for pool: %v, err: %v", p.Address, err)
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:     rpcData.liquidityState.BaseL,
		ReinvestL:     rpcData.liquidityState.ReinvestL,
		ReinvestLLast: rpcData.liquidityState.ReinvestLLast,
		SqrtPriceX96:  rpcData.poolState.SqrtP,
		Tick:          rpcData.poolState.CurrentTick,
		Ticks:         ticks,
	})
	if err != nil {
		logger.Errorf("failed to marshal extra data for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.reserve0.String(),
		rpcData.reserve1.String(),
	}

	logger.Infof("[Elastic] Finish getting new state of pool: %v", p.Address)

	return p, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		liquidityState LiquidityState
		poolState      PoolState
		reserve0       = zeroBI
		reserve1       = zeroBI
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    elasticPoolABI,
		Target: p.Address,
		Method: methodGetLiquidityState,
		Params: nil,
	}, []interface{}{&liquidityState})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    elasticPoolABI,
		Target: p.Address,
		Method: methodGetPoolState,
		Params: nil,
	}, []interface{}{&poolState})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&reserve1})
	}

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.Errorf("failed to process tryAggregate for pool: %v, err: %v", p.Address, err)
		return FetchRPCResult{}, err
	}

	return FetchRPCResult{
		liquidityState: liquidityState,
		poolState:      poolState,
		reserve0:       reserve0,
		reserve1:       reserve1,
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(
			fmt.Sprintf(`{
				pool(id: "%v") {
					id
					ticks(orderBy: tickIdx, orderDirection: asc, first: %v, skip: %v) {
						tickIdx
						liquidityNet
						liquidityGross
					}
				}
				_meta { block { timestamp }}
			}`, poolAddress, graphFirstLimit, skip),
		)

		var resp struct {
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			logger.Errorf("failed to query subgraph for pool: %v, err: %v", poolAddress, err)
			return nil, err
		}

		resp.Meta.CheckIsLagging(d.config.DexID, poolAddress)

		if resp.Pool == nil || len(resp.Pool.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Pool.Ticks...)

		if len(resp.Pool.Ticks) < graphFirstLimit {
			break
		}

		skip += len(resp.Pool.Ticks)
		if skip > graphSkipLimit {
			logger.Infoln("hit skip limit, continue in next cycle")
			break
		}
	}

	return ticks, nil
}
