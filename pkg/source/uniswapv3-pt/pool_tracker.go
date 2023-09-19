package uniswapv3pt

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"
)

type PoolTracker struct {
	config          *Config
	ethrpcClient    *ethrpc.Client
	poolTicksClient *PoolTicksClient
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	initializedCfg, err := initializeConfig(cfg)
	if err != nil {
		return nil, err
	}

	poolTicksClient, err := NewPoolTicksClient(cfg.PoolTicksAPI, nil)
	if err != nil {
		return nil, err
	}

	return &PoolTracker{
		config:          initializedCfg,
		ethrpcClient:    ethrpcClient,
		poolTicksClient: poolTicksClient,
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
		logger.WithFields(logger.Fields{"error": err}).Errorf("failed to parse pools")
		return nil, err
	}

	logger.WithFields(logger.Fields{
		"filePath": cfg.PreGenesisPoolPath,
		"numPools": len(pools),
	}).Info("Success get pools from file")

	for _, p := range pools {
		cfg.preGenesisPoolIDs = append(cfg.preGenesisPoolIDs, p.ID)
	}

	return cfg, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexID":       d.config.DexID,
		"poolAddress": p.Address,
	}).Info("Start getting new state of pool")

	var (
		rpcData   FetchRPCResult
		poolTicks []Tick
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
			poolTicks, err = d.getPoolTicks(p.Address)
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
		}).Errorf("Failed to fetch pool state")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcData.liquidity,
		SqrtPriceX96: rpcData.slot0.SqrtPriceX96,
		Tick:         rpcData.slot0.Tick,
		Ticks:        poolTicks,
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

	logger.WithFields(logger.Fields{
		"dexID":       d.config.DexID,
		"poolAddress": p.Address,
	}).Info("Finish updating state of pool")

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

func (d *PoolTracker) getPoolTicks(poolAddress string) ([]Tick, error) {
	poolTicks, err := d.poolTicksClient.GetPoolTicks(poolAddress)
	if err != nil {
		return nil, err
	}

	ticks := make([]Tick, 0, len(poolTicks))
	for _, tick := range poolTicks {
		ticks = append(ticks, Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	return ticks, nil
}
