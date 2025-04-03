package bunniv2

import (
	"context"
	"encoding/hex"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func GetCustomReserves(ctx context.Context, p entity.Pool, ethrpcClient *ethrpc.Client) (entity.PoolReserves, error) {
	poolIDBytesSlice, err := hex.DecodeString(p.Address[2:])
	if err != nil {
		return nil, err
	}

	var poolIDBytes [32]byte
	copy(poolIDBytes[:], poolIDBytesSlice)

	hubCaller, err := NewBunniV2HubContractCaller(HubAddress, ethrpcClient.GetETHClient())
	if err != nil {
		return nil, err
	}

	poolState, err := hubCaller.PoolState(nil, poolIDBytes)
	if err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Reserve0.Add(poolState.Reserve0, poolState.RawBalance0).String(),
		poolState.Reserve1.Add(poolState.Reserve1, poolState.RawBalance1).String(),
	}, nil
}
