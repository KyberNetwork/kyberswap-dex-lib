package traderjoev21

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
	logger.Infof("[TraderJoe v2.1] Start getting new state of pool: %v", p.Address)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var reserves Reserves

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetReservesMethod,
	}, []interface{}{&reserves})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.Errorf("failed to call pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.ReserveX.String(),
		reserves.ReserveY.String(),
	}

	logger.Infof("[TraderJoe v2.1] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
