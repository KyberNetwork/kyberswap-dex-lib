package solidlyv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":     p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pool state")
	}()

	var staticExtra velodromev2.PoolStaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	reserveData, blockNumber, err := d.getReserves(ctx, p.Address)
	if err != nil {
		return p, err
	}

	isPaused, fee, err := d.getFactoryData(ctx, p.Address, blockNumber)
	if err != nil {
		return p, err
	}

	return d.updatePool(p, reserveData, isPaused, fee, blockNumber.Uint64())
}

func (d *PoolTracker) getFactoryData(
	ctx context.Context,
	poolAddress string,
	blockNumber *big.Int,
) (bool, uint64, error) {
	var (
		isPaused bool
		fee      *big.Int
	)

	getFactoryDataRequest := d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFactoryDataRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&isPaused})
	getFactoryDataRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodFeeRatio,
		Params: []interface{}{},
	}, []interface{}{&fee})

	if _, err := getFactoryDataRequest.TryBlockAndAggregate(); err != nil {
		return false, 0, err
	}

	return isPaused, fee.Uint64(), nil
}

func (d *PoolTracker) updatePool(
	pool entity.Pool,
	reserveData velodromev2.ReserveData,
	isPaused bool,
	fee uint64,
	blockNumber uint64) (entity.Pool, error) {
	if pool.BlockNumber > blockNumber {
		return pool, nil
	}

	poolExtra := velodromev2.PoolExtra{
		IsPaused: isPaused,
		Fee:      fee,
	}
	poolExtraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	pool.Extra = string(poolExtraBytes)
	pool.BlockNumber = blockNumber
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string) (velodromev2.ReserveData, *big.Int, error) {
	var getReservesResult velodromev2.GetReservesResult

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return velodromev2.ReserveData{}, nil, err
	}

	return velodromev2.ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}, resp.BlockNumber, nil
}
