package uniswapv3

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/samber/lo"
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
	initializedCfg, err := initializeConfig(cfg)
	if err != nil {
		return nil, err
	}

	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolTracker{
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
		}).Errorf("failed to parse pools")
		return nil, err
	}

	logger.Infof("got %v pools from file: %s", len(pools), cfg.PreGenesisPoolPath)

	for _, p := range pools {
		cfg.preGenesisPoolIDs = append(cfg.preGenesisPoolIDs, p.ID)
	}

	return cfg, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool: %v", d.config.DexID, p.Address)

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
		// Ad-hoc logic to handle edge case on Optimism
		// Link to issue: https://www.notion.so/kybernetwork/Aggregator-1-20-defect-1caec6062f9d4da0918fc3443e6e1963#0810d1462cc14f0a9465f935c9e641fe
		// TLDR: Optimism has some pre-genesis Uniswap V3 pool. Subgraph does not have data for these pools
		// So we have to fetch ticks data from the TickLens smart contract (which is slower).
		if lo.Contains[string](d.config.preGenesisPoolIDs, p.Address) {
			poolTicks, err = d.getPoolTicksFromSC(ctx, p)
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Errorf("failed to call SC for pool ticks")
			}
		} else {
			// If pool is not pre-genesis, fetch from subgraph
			poolTicks, err = d.getPoolTicks(ctx, p.Address)
			if err != nil {
				logger.WithFields(logger.Fields{
					"poolAddress": p.Address,
					"error":       err,
				}).Errorf("failed to query subgraph for pool ticks")
			}
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
		Liquidity:    rpcData.liquidity,
		SqrtPriceX96: rpcData.slot0.SqrtPriceX96,
		Tick:         rpcData.slot0.Tick,
		Ticks:        ticks,
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

	logger.Infof("[%s] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		liquidity *big.Int
		slot0     Slot0
		reserve0  = zeroBI
		reserve1  = zeroBI
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
		Params: nil,
	}, []interface{}{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetSlot0,
		Params: nil,
	}, []interface{}{&slot0})

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
		slot0:     slot0,
		reserve0:  reserve0,
		reserve1:  reserve1,
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()
	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
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
