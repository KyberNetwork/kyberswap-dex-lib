package velodromev1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   uniswapv2.ILogDecoder
		feeTracker   IFeeTracker
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	tracker := &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   uniswapv2.NewLogDecoder(),
	}
	if feeTrackerCfg := config.FeeTracker; feeTrackerCfg != nil {
		tracker.feeTracker = NewGenericFeeTracker(ethrpcClient, feeTrackerCfg)
	}
	return tracker, nil
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

	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	reserveData, blockNumber, err := d.getReserves(ctx, p.Address, params.Logs)
	if err != nil {
		return p, err
	}

	isPaused, err := d.getFactoryData(ctx, blockNumber)
	if err != nil {
		return p, err
	}

	fee, err := d.getFee(ctx, p.Address, staticExtra.Stable, blockNumber)
	if err != nil {
		return p, err
	}

	return d.updatePool(p, reserveData, isPaused, fee, blockNumber)
}

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, *big.Int,
	error) {
	reserveData, blockNumber, err := d.getReservesFromLogs(logs)
	if err != nil {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	if reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getFactoryData(ctx context.Context, blockNumber *big.Int) (bool, error) {
	var isPaused bool

	getFactoryDataRequest := d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFactoryDataRequest.AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: d.config.FactoryAddress,
		Method: pairFactoryMethodIsPaused,
		Params: nil,
	}, []any{&isPaused})

	if _, err := getFactoryDataRequest.TryBlockAndAggregate(); err != nil {
		return false, err
	}

	return isPaused, nil
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string, isStable bool, blockNumber *big.Int) (uint64,
	error) {
	feeTracker := d.feeTracker
	if feeTracker == nil {
		return d.config.FeePrecision / 100, nil
	}
	return feeTracker.GetFee(ctx, poolAddress, d.config.FactoryAddress, isStable, blockNumber)
}

func (d *PoolTracker) updatePool(
	pool entity.Pool,
	reserveData ReserveData,
	isPaused bool,
	fee uint64,
	blockNumber *big.Int) (entity.Pool, error) {
	if pool.BlockNumber > blockNumber.Uint64() {
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
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, *big.Int, error) {
	var getReservesResult GetReservesResult

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []any{&getReservesResult})

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}

	return ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) getReservesFromLogs(logs []types.Log) (ReserveData, *big.Int, error) {
	if len(logs) == 0 {
		return ReserveData{}, nil, nil
	}

	if d.logDecoder == nil {
		return ReserveData{}, nil, nil
	}

	return d.logDecoder.Decode(logs)
}
