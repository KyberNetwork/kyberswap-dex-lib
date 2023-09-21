package biswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[%s] Start getting new state of pool", d.config.DexID)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var (
		reserves Reserves
		swapFee  uint32
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    biswapPairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    biswapPairABI,
		Target: p.Address,
		Method: pairMethodGetSwapFee,
		Params: nil,
	}, []interface{}{&swapFee})

	resp, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process tryAggregate for pool")
		return entity.Pool{}, err
	}

	if len(resp.Result) != 2 {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("result of tryAggregate for pool: %v is broken", p.Address)
		return entity.Pool{}, err
	}

	if !resp.Result[0] || !resp.Result[1] {
		logger.Warnf("failed to fetch pool state, reserves: %v, swapFee: %v", resp.Result[0], resp.Result[1])
		return entity.Pool{}, err
	}

	swapFeeFL := float64(swapFee) / float64(d.config.FeePrecision)
	p.SwapFee = swapFeeFL
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.Reserve0.String(),
		reserves.Reserve1.String(),
	}

	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[%s] Finish getting new state of pool", d.config.DexID)

	return p, nil

}
