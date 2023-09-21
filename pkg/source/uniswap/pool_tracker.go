package uniswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Uniswap V2] Start getting new state of pool: %v", p.Address)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var reserves Reserves

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.Errorf("failed to process tryAggregate for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.Reserve0.String(),
		reserves.Reserve1.String(),
	}

	logger.Infof("[Uniswap V2] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
