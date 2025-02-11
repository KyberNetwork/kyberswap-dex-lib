package solidlyv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
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

	if d.config.IsMemecoreDEX {
		return d.updateMemecorePool(ctx, p, overrides)
	}

	return d.updateStandardPool(ctx, p, overrides)
}

func (d *PoolTracker) updateMemecorePool(
	ctx context.Context,
	pool entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	var (
		poolFee           uint16
		getReservesResult MemecoreReserves
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    memecoreABI,
		Target: pool.Address,
		Method: memecoreMethodPoolFee,
		Params: []interface{}{},
	}, []interface{}{&poolFee})
	req.AddCall(&ethrpc.Call{
		ABI:    memecoreABI,
		Target: pool.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	reserves := velodromev2.ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}

	poolExtra := velodromev2.PoolExtra{
		Fee: uint64(poolFee),
	}

	return d.updatePool(pool, reserves, poolExtra, resp.BlockNumber.Uint64())
}

func (d *PoolTracker) updateStandardPool(
	ctx context.Context,
	pool entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	var (
		isPaused          bool
		fee               *big.Int
		getReservesResult velodromev2.GetReservesResult
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&isPaused})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: pool.Address,
		Method: poolMethodFeeRatio,
		Params: []interface{}{},
	}, []interface{}{&fee})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: pool.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	reserves := velodromev2.ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}

	poolExtra := velodromev2.PoolExtra{
		IsPaused: isPaused,
		Fee:      fee.Uint64(),
	}

	return d.updatePool(pool, reserves, poolExtra, resp.BlockNumber.Uint64())
}

func (d *PoolTracker) updatePool(
	pool entity.Pool,
	reserveData velodromev2.ReserveData,
	poolExtra velodromev2.PoolExtra,
	blockNumber uint64) (entity.Pool, error) {
	if pool.BlockNumber > blockNumber {
		return pool, nil
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
