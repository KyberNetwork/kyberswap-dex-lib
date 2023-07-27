package algebrav1

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
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

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[%v] Start getting new state of pool: %v", d.config.DexID, p.Address)

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p)
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
		// If pool is not pre-genesis, fetch from subgraph
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
		Liquidity: rpcData.liquidity,
		GlobalState: rpcData.state,
		Ticks:     ticks,
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
	p.Reserves = entity.PoolReserves{
		rpcData.reserve0.String(),
		rpcData.reserve1.String(),
	}

	logger.Infof("[%v] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		liquidity *big.Int
		state     GlobalState
		reserve0  = bignumber.ZeroBI
		reserve1  = bignumber.ZeroBI
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
		Params: nil,
	}, []interface{}{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetGlobalState,
		Params: nil,
	}, []interface{}{&state})

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
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process tryAggregate")
		return FetchRPCResult{}, err
	}

	return FetchRPCResult{
		liquidity: liquidity,
		state:     state,
		reserve0:  reserve0,
		reserve1:  reserve1,
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	allowSubgraphError := d.config.AllowSubgraphError
	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks `json:"pool"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Pool == nil {
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
