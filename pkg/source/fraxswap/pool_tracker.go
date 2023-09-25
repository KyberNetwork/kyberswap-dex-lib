package fraxswap

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})
	log.Infof("[Fraxswap] Start updating state ...")

	var reserveAfterTwammOutput ReserveAfterTwammOutput
	var feeOutput FeeOutput

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: poolMethodGetReserveAfterTwamm,
		Params: []interface{}{big.NewInt(time.Now().Unix())},
	}, []interface{}{&reserveAfterTwammOutput})

	calls.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&feeOutput})

	if _, err := calls.TryAggregate(); err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Errorf("[Fraxswap] failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	extra := Extra{
		Reserve0: reserveAfterTwammOutput.Reserve0,
		Reserve1: reserveAfterTwammOutput.Reserve1,
		Fee:      feeOutput.Fee,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserveAfterTwammOutput.Reserve0.String(), reserveAfterTwammOutput.Reserve1.String()}
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)

	log.Infof("[Fraxswap] Finish getting new state of pool")

	return p, nil
}
