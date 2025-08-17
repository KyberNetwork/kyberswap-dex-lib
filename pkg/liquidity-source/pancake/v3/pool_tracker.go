package pancakev3

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

	*pooltrack.InactivePoolTracker
}

var _ = pooltrack.RegisterFactoryCEG(DexTypePancakeV3, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:              cfg,
		ethrpcClient:        ethrpcClient,
		graphqlClient:       graphqlClient,
		InactivePoolTracker: pooltrack.NewInactivePoolTracker(cfg.TrackInactivePools),
	}, nil
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

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	blockNumber, err := d.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

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
		Liquidity:         rpcData.Liquidity,
		SqrtPriceX96:      rpcData.Slot0.SqrtPriceX96,
		TickSpacing:       rpcData.TickSpacing.Uint64(),
		Tick:              rpcData.Slot0.Tick,
		Ticks:             ticks,
		ObsBlockTimestamp: rpcData.ObsBlockTimestamp,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = int64(rpcData.ObsBlockTimestamp)
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = blockNumber
	p.IsInactive = d.IsInactive(&p, time.Now().Unix())

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	var (
		liquidity    *big.Int
		slot0        Slot0
		tickSpacing  *big.Int
		reserve0     = zeroBI
		reserve1     = zeroBI
		observations Observations
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pancakeV3PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
	}, []any{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pancakeV3PoolABI,
		Target: p.Address,
		Method: methodGetSlot0,
	}, []any{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pancakeV3PoolABI,
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

	_, err = d.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    pancakeV3PoolABI,
			Target: p.Address,
			Method: methodObservations,
			Params: []any{big.NewInt(int64(slot0.ObservationIndex))},
		}, []any{&observations}).Call()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get observations")
		return nil, err
	}

	return &FetchRPCResult{
		Liquidity:         liquidity,
		Slot0:             slot0,
		TickSpacing:       tickSpacing,
		Reserve0:          reserve0,
		Reserve1:          reserve1,
		ObsBlockTimestamp: observations.BlockTimestamp,
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
			Ticks []TickResp                `json:"ticks"`
			Meta  *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError && resp.Ticks == nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to query subgraph")
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
