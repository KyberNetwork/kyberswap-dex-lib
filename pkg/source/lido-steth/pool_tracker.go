package lido_steth

import (
	"context"
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

	log.Infof("[Lido-stETH] Start getting new pool's state")

	reserves, err := d.getPoolReserves(ctx, p)
	if err != nil {
		log.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getPoolReserves")
		return entity.Pool{}, err
	}

	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()

	log.Infof("[Lido-stETH] Finish getting new state of pool")

	return p, nil
}

func (d *PoolTracker) getPoolReserves(ctx context.Context, p entity.Pool) (entity.PoolReserves, error) {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var totalPooledEther *big.Int
	var totalShares *big.Int

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stEthABI,
		Target: p.Address,
		Method: methodTotalPooledEther,
		Params: nil,
	}, []interface{}{&totalPooledEther})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stEthABI,
		Target: p.Address,
		Method: methodTotalShares,
		Params: nil,
	}, []interface{}{&totalShares})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to process tryAggregate")
		return nil, err
	}

	return entity.PoolReserves{totalPooledEther.String(), totalShares.String()}, nil
}
