package velodromev2

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

	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	reserveData, isPaused, fee, blockNumber, err := d.getPoolData(ctx, p.Address, staticExtra.Stable, overrides)
	if err != nil {
		return p, err
	}

	return d.updatePool(p, reserveData, isPaused, fee, blockNumber)
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress string,
	stable bool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (ReserveData, bool, uint64, uint64, error) {
	var (
		isPaused          bool
		fee               *big.Int
		getReservesResult GetReservesResult
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolFactoryABI,
		Target: d.config.FactoryAddress,
		Method: poolFactoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&isPaused})
	req.AddCall(&ethrpc.Call{
		ABI:    poolFactoryABI,
		Target: d.config.FactoryAddress,
		Method: poolFactoryMethodGetFee,
		Params: []interface{}{common.HexToAddress(poolAddress), stable},
	}, []interface{}{&fee})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, false, 0, 0, err
	}

	return ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}, isPaused, fee.Uint64(), resp.BlockNumber.Uint64(), nil
}

func (d *PoolTracker) updatePool(
	pool entity.Pool,
	reserveData ReserveData,
	isPaused bool,
	fee uint64,
	blockNumber uint64) (entity.Pool, error) {
	if pool.BlockNumber > blockNumber {
		return pool, nil
	}

	poolExtra := PoolExtra{
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
