package iziswap

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/izumiFinance/iZiSwap-SDK-go/swap"
	"github.com/sourcegraph/conc/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	if cfg.PointRange <= 0 {
		cfg.PointRange = DEFAULT_PT_RANGE
	}
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[iZiSwap] Start getting new state of pool: %v", p.Address)

	g := pool.New().WithContext(ctx)

	rpcData, err := d.fetchPoolState(ctx, p)

	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to fetch pool state, pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	swapFee := int(p.SwapFee)
	pointDelta := getPointDelta(swapFee)
	rightMostPt := RIGHT_MOST_PT / pointDelta * pointDelta
	leftMostPt := -rightMostPt

	poolInfo := swap.PoolInfo{
		CurrentPoint: int(rpcData.state.CurrentPoint.Int64()),
		Fee:          swapFee,
		PointDelta:   pointDelta,
		RightMostPt:  rightMostPt,
		LeftMostPt:   leftMostPt,
		Liquidity:    rpcData.state.Liquidity,
		LiquidityX:   rpcData.state.LiquidityX,
	}

	var (
		liquidityPointData  []swap.LiquidityPoint
		limitOrderPointData []swap.LimitOrderPoint
	)

	g.Go(func(context.Context) error {
		var err error
		liquidityPointData, err = d.getLiquiditySnapshot(ctx, p, poolInfo)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to call SC for pool liquidity snapshot")
		}
		return err
	})
	g.Go(func(context.Context) error {
		var err error
		limitOrderPointData, err = d.getLimitOrderSnapshot(ctx, p, poolInfo)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to call SC for pool limitOrder snapshot")
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
	poolInfo.Liquidities = liquidityPointData
	poolInfo.LimitOrders = limitOrderPointData

	extraBytes, err := json.Marshal(poolInfo)
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

	logger.Infof("[iZiSwap] Finish updating state of pool: %v", p.Address)

	return p, nil
}

func (d *PoolTracker) fetchPoolState(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		state    State
		reserve0 = zeroBI
		reserve1 = zeroBI
	)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    iZiSwapPoolABI,
		Target: p.Address,
		Method: methodGetState,
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
		state:    state,
		reserve0: reserve0,
		reserve1: reserve1,
	}, err

}
