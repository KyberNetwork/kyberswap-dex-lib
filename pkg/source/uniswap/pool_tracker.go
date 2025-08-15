package uniswap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexTypeUniswap, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Uniswap V2] Start getting new state of pool: %v", p.Address)

	var (
		err      error
		reserves Reserves
	)

	latestSyncEvent := findLatestSyncEvent(params.Logs)
	if latestSyncEvent == nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Info("Fetch reserves from node")
		reserves, err = d.fetchReservesFromNode(ctx, p.Address)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Error("Fail to fetch reserves from node")
		} else {
			p.Timestamp = int64(reserves.BlockTimestampLast)
		}
	} else {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"event":       latestSyncEvent,
		}).Debug("Decode sync event")
		reserves, err = decodeSyncEvent(*latestSyncEvent)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"event":       latestSyncEvent,
				"error":       err,
			}).Error("Fail to decode sync event")
		} else {
			p.Timestamp = int64(params.BlockHeaders[latestSyncEvent.BlockNumber].Timestamp)
		}
	}

	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		reserveString(reserves.Reserve0),
		reserveString(reserves.Reserve1),
	}
	p.IsInactive = d.IsInactive(&p, time.Now().Unix())

	logger.Infof("[Uniswap V2] Finish getting new state of pool: %v", p.Address)

	return p, nil
}

func (d *PoolTracker) fetchReservesFromNode(ctx context.Context, poolAddress string) (Reserves, error) {
	var reserves Reserves

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []interface{}{&reserves})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.Errorf("failed to process tryAggregate for pool: %v, err: %v", poolAddress, err)
		return Reserves{}, err
	}

	return reserves, nil
}

func reserveString(reserve *big.Int) string {
	if reserve == nil {
		return reserveZero
	}
	return reserve.String()
}

func (d *PoolTracker) IsInactive(p *entity.Pool, currentTimestamp int64) bool {
	if !d.config.TrackInactivePools.Enabled {
		return false
	}

	inactiveTimeThresholdInSecond := int64(d.config.TrackInactivePools.TimeThreshold.Seconds())
	if inactiveTimeThresholdInSecond <= 0 {
		return false
	}

	return currentTimestamp-p.Timestamp > inactiveTimeThresholdInSecond
}

func (d *PoolTracker) GetInactivePools(_ context.Context, currentTimestamp int64, pools ...entity.Pool) ([]string, error) {
	if len(pools) == 0 {
		return nil, nil
	}

	inactivePools := lo.Filter(pools, func(p entity.Pool, _ int) bool {
		return d.IsInactive(&p, currentTimestamp)
	})

	return lo.Map(inactivePools, func(p entity.Pool, _ int) string { return p.Address }), nil
}
