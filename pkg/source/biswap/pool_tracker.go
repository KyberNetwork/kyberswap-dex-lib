package biswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Biswap] Start getting new state of pool")

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

	swapFeeFL := float64(swapFee) / 1000
	p.SwapFee = swapFeeFL
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.Reserve0.String(),
		reserves.Reserve1.String(),
	}

	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Biswap] Finish getting new state of pool")

	return p, nil

}
