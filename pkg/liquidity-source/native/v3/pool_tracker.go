package v3

import (
	"context"
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
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

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

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		return nil, err
	}

	var (
		slot0                  Slot0
		liquidity, tickSpacing *big.Int

		reserves = [2]*big.Int{Zero, Zero}
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodGetLiquidity,
	}, []any{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodGetSlot0,
	}, []any{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodTickSpacing,
	}, []any{&tickSpacing})

	var underlyingTokens = make([]common.Address, len(p.Tokens))
	var needFetchUnderlyingToken = len(staticExtra.UnderlyingTokens) == 0

	for i := range len(p.Tokens) {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[i].Address,
			Method: erc20MethodBalanceOf,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserves[i]})

		if needFetchUnderlyingToken {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: p.Tokens[i].Address,
				Method: lpTokenMethodUnderlying,
				Params: nil,
			}, []any{&underlyingTokens[i]})
		}
	}

	res, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process TryBlockAndAggregate")
		return nil, err
	}

	if needFetchUnderlyingToken {
		staticExtra.UnderlyingTokens = [2]string{
			underlyingTokens[0].Hex(),
			underlyingTokens[1].Hex(),
		}
	}

	return &FetchRPCResult{
		Liquidity:        liquidity,
		Slot0:            slot0,
		Reserves:         reserves,
		TickSpacing:      staticExtra.TickSpacing,
		UnderlyingTokens: staticExtra.UnderlyingTokens,
		BlockNumber:      res.BlockNumber.Uint64(),
	}, nil
}

func (d *PoolTracker) fetchPoolTicks(ctx context.Context, p entity.Pool, _ sourcePool.GetNewPoolStateParams) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	poolTicks, err := d.getPoolTicks(ctx, p.Address)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to query subgraph for pool ticks")
		return nil, err
	}
	return poolTicks, nil
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
		poolTicks, err = d.fetchPoolTicks(ctx, p, param)
		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	ticks := lo.Map(poolTicks, func(tickResp TickResp, _ int) Tick {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			return Tick{}
		}
		return tick
	})

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcData.Liquidity,
		Unlocked:     rpcData.Slot0.Unlocked,
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

	staticExtraBytes, err := json.Marshal(StaticExtra{
		TickSpacing:      rpcData.TickSpacing,
		UnderlyingTokens: rpcData.UnderlyingTokens,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal static extra data")
		return entity.Pool{}, err
	}

	if rpcData.Slot0.Unlocked {
		p.Reserves = entity.PoolReserves{
			rpcData.Reserves[0].String(),
			rpcData.Reserves[1].String(),
		}
	} else {
		p.Reserves = entity.PoolReserves{"0", "0"}
	}

	p.StaticExtra = string(staticExtraBytes)
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = rpcData.BlockNumber

	l.Infof("Finish updating state of pool")

	return p, nil
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

		var response struct {
			Ticks []TickResp `json:"ticks"`
		}

		if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError && len(response.Ticks) > 0 {
				ticks = append(ticks, response.Ticks...)
				break
			}

			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph")
			return nil, err
		}

		if len(response.Ticks) == 0 {
			break
		}

		ticks = append(ticks, response.Ticks...)
		lastTickIdx = response.Ticks[len(response.Ticks)-1].TickIdx

		if len(response.Ticks) < graphFirstLimit {
			break
		}
	}

	return ticks, nil
}
