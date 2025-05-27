package ramsesv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexTypeRamsesV2, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {

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
		poolTicks []ticklens.TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.FetchRPCData(ctx, p, 0)
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
		Unlocked:     rpcData.Slot0.Unlocked,
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
	p.BlockNumber = rpcData.BlockNumber
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}

	logger.Infof("[%s] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := d.FetchRPCData(ctx, p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p entity.Pool, blockNumber uint64) (FetchRPCResult, error) {
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

	if d.config.IsPoolV3 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    ramsesV3PoolABI,
			Target: p.Address,
			Method: methodV3Fee,
			Params: nil,
		}, []interface{}{&feeTier})
	} else {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    ramsesV2PoolABI,
			Target: p.Address,
			Method: methodV2CurrentFee,
			Params: nil,
		}, []interface{}{&feeTier})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodV2GetLiquidity,
		Params: nil,
	}, []interface{}{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodV2GetSlot0,
		Params: nil,
	}, []interface{}{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    ramsesV2PoolABI,
		Target: p.Address,
		Method: methodV2TickSpacing,
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

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process Aggregate")
		return FetchRPCResult{}, err
	}

	return FetchRPCResult{
		Liquidity:   liquidity,
		Slot0:       slot0,
		FeeTier:     feeTier.Int64(),
		TickSpacing: tickSpacing.Uint64(),
		Reserve0:    reserve0,
		Reserve1:    reserve1,
		BlockNumber: resp.BlockNumber.Uint64(),
	}, err
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]ticklens.TickResp, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []ticklens.TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []ticklens.TickResp       `json:"ticks"`
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
