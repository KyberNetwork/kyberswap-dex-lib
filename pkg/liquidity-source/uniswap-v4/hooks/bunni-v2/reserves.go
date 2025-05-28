package bunniv2

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func GetCustomReserves(ctx context.Context, p entity.Pool, ethrpcClient *ethrpc.Client) (entity.PoolReserves, error) {
	hubCaller, err := NewBunniV2HubContractCaller(HubAddress, ethrpcClient.GetETHClient())
	if err != nil {
		return nil, err
	}

	poolState, err := hubCaller.PoolState(&bind.CallOpts{Context: ctx}, common.HexToHash(p.Address))
	if err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Reserve0.Add(poolState.Reserve0, poolState.RawBalance0).String(),
		poolState.Reserve1.Add(poolState.Reserve1, poolState.RawBalance1).String(),
	}, nil
}
