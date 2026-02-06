package someswapv1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	var (
		reserves    ReserveData
		blockNumber *big.Int
		err         error
	)

	latestSyncEvent := findLatestSyncEvent(params.Logs)
	if latestSyncEvent != nil {
		reserves, err = decodeSyncEvent(*latestSyncEvent)
		blockNumber = new(big.Int).SetUint64(latestSyncEvent.BlockNumber)
	} else {
		reserves, blockNumber, err = d.getReservesFromRPCNode(ctx, p.Address)
	}
	if err != nil {
		return p, err
	}

	if blockNumber != nil && p.BlockNumber > blockNumber.Uint64() {
		return p, nil
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{reserveString(reserves.Reserve0), reserveString(reserves.Reserve1)}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	logger.WithFields(
		logger.Fields{
			"pool_id":     p.Address,
			"old_reserve": p.Reserves,
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting new pool state")

	return p, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, *big.Int, error) {
	var reserves ReserveData
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []any{&reserves})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}
	return reserves, resp.BlockNumber, nil
}

func reserveString(reserve *big.Int) string {
	if reserve == nil {
		return "0"
	}
	return reserve.String()
}
