package v3

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/machinebox/graphql"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
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
	graphqlClient := graphqlpkg.New(graphqlpkg.Config{
		Url:     cfg.SubgraphAPI,
		Header:  cfg.SubgraphHeaders,
		Timeout: graphQLRequestTimeout,
	})

	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool: %v", d.config.DexID, p.Address)

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p, 0)
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
		if d.config.AlwaysUseTickLens {
			poolTicks, err = ticklens.GetPoolTicksFromSC(ctx, d.ethrpcClient, d.config.TickLensAddress, p, param)
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

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
		Liquidity:    rpcData.Liquidity,
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		FeeTier:      rpcData.FeeTier,
		TickSpacing:  rpcData.TickSpacing,
		Tick:         rpcData.Slot0.Tick,
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
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}

	logger.Infof("[%s] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := d.fetchRPCData(ctx, p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool, blockNumber uint64) (FetchRPCResult, error) {
	var (
		liquidity   *big.Int
		slot0       Slot0
		feeTier     *big.Int
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
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
		Params: nil,
	}, []interface{}{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodGetSlot0,
		Params: nil,
	}, []interface{}{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodCurrentFee,
		Params: nil,
	}, []interface{}{&feeTier})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodTickSpacing,
		Params: nil,
	}, []interface{}{&tickSpacing})

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
		Liquidity:   liquidity,
		Slot0:       slot0,
		FeeTier:     feeTier.Int64(),
		TickSpacing: tickSpacing.Uint64(),
		Reserve0:    reserve0,
		Reserve1:    reserve1,
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []TickResp

	for {
		req := graphql.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []TickResp                `json:"ticks"`
			Meta  *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Ticks == nil {
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
