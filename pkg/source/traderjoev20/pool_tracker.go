package traderjoev20

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoecommon"
)

type PoolTracker struct {
	*traderjoecommon.PoolTracker[Reserves]
}

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		PoolTracker: &traderjoecommon.PoolTracker[Reserves]{
			EthrpcClient:          ethrpcClient,
			PairABI:               pairABI,
			PairGetReservesMethod: pairGetReservesMethod,
		},
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[TraderJoe v2.0] Start getting new state of pool: %v", p.Address)
	p, err := d.PoolTracker.GetNewPoolState(ctx, p)
	logger.Infof("[TraderJoe v2.0] Finish getting new state of pool: %v", p.Address)
	return p, err
}
