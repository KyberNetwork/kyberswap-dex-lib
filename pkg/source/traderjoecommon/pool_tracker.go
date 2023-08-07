package traderjoecommon

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type Reserves interface {
	GetPoolReserves() entity.PoolReserves
}

type PoolTracker[R Reserves] struct {
	EthrpcClient *ethrpc.Client

	PairABI               abi.ABI
	PairGetReservesMethod string
}

func (d *PoolTracker[R]) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var reserves R
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    d.PairABI,
		Target: p.Address,
		Method: d.PairGetReservesMethod,
	}, []interface{}{&reserves})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.Errorf("failed to call pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves.GetPoolReserves()

	return p, nil
}
