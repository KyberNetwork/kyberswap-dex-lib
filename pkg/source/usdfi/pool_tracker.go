package usdfi

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

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})
	log.Infof("[Usdfi] Start updating state ...")

	var reserve Reserves

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserve})

	if _, err := calls.TryAggregate(); err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Errorf("[Usdfi] failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserve.Reserve0.String(), reserve.Reserve1.String()}
	p.Timestamp = time.Now().Unix()

	log.Infof("[Usdfi] Finish getting new state of pool")

	return p, nil
}
