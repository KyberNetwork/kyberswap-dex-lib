package crowdswapv2

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
	logger.Infof("[Crowdswap V2] Start getting new state of pool: %v", p.Address)

	var (
		reserves Reserves
		swapFee  uint8
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    crowdswapV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    crowdswapV2PairABI,
		Target: p.Address,
		Method: pairMethodGetSwapFee,
		Params: nil,
	}, []interface{}{&swapFee})

	resp, err := calls.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Crowdswap V2]: failed to process tryAggregate for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	if len(resp.Result) != 2 {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("[Crowdswap V2]: result of tryAggregate for pool: %v is broken", p.Address)
		return entity.Pool{}, err
	}

	if !resp.Result[0] || !resp.Result[1] {
		logger.Warnf("[Crowdswap V2]: failed to fetch pool state, reserves: %v, swapFee: %v", resp.Result[0], resp.Result[1])
		return entity.Pool{}, err
	}

	var swapFeeFL float64 = float64(swapFee) / bps
	p.SwapFee = swapFeeFL
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.Reserve0.String(),
		reserves.Reserve1.String(),
	}

	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Crowdswap V2] Finish getting new state of pool")

	return p, nil
}
