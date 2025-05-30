package uniswapv3

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexTypeUniswapV3, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	initializedCfg, err := initializeConfig(cfg)
	if err != nil {
		return nil, err
	}

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
		}).Error("failed to parse pools")
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
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	l.Info("Start getting new state of pool")

	blockNumber, err := d.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.FetchRPCData(ctx, &p, 0)
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
		if d.config.AlwaysUseTickLens || lo.Contains[string](d.config.preGenesisPoolIDs, p.Address) {
			poolTicks, err = ticklens.GetPoolTicksFromSC(ctx, d.ethrpcClient, d.config.TickLensAddress, p, param)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
		} else {
			// If pool is not pre-genesis, fetch from subgraph
			poolTicks, err = d.getPoolTicks(ctx, p.Address)
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

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcData.Liquidity,
		TickSpacing:  rpcData.TickSpacing.Uint64(),
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
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = blockNumber

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	var (
		liquidity   *big.Int
		slot0       Slot0
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

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
	}, []any{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetSlot0,
	}, []any{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV3PoolABI,
		Target: p.Address,
		Method: methodTickSpacing,
	}, []any{&tickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20MethodBalanceOf,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20MethodBalanceOf,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve1})
	}

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return nil, err
	}

	return &FetchRPCResult{
		Liquidity:   liquidity,
		Slot0:       slot0,
		TickSpacing: tickSpacing,
		Reserve0:    reserve0,
		Reserve1:    reserve1,
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	allowSubgraphError := d.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []TickResp `json:"ticks"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
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
