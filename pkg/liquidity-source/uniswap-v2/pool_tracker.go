package uniswapv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   ILogDecoder
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
		logDecoder:   NewLogDecoder(),
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

	reserveData, blockNumber, err := d.getReserves(ctx, p.Address, params.Logs)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	fee, err := d.getFee(ctx, p.Address, blockNumber)
	if err != nil {
		return p, err
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      reserveData,
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, reserveData, fee, blockNumber)
}

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, *big.Int,
	error) {
	reserveData, blockNumber, err := d.getReservesFromLogs(logs)
	if err != nil || reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}
	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string, blockNumber *big.Int) (uint64, error) {
	feeTracker := d.feeTracker
	if feeTracker == nil {
		return d.config.Fee, nil
	}
	return feeTracker.GetFee(ctx, poolAddress, d.config.FactoryAddress, blockNumber)
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData, fee uint64,
	blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		Fee:          fee,
		FeePrecision: d.config.FeePrecision,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	pool.Extra = string(extraBytes)
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, *big.Int, error) {
	var getReservesResult ReserveData

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	if d.config.OldReserveMethods {
		getReservesRequest.AddCall(&ethrpc.Call{
			ABI:    uniswapV2PairABI,
			Target: poolAddress,
			Method: pairMethodReserve0,
		}, []any{&getReservesResult.Reserve0}).AddCall(&ethrpc.Call{
			ABI:    uniswapV2PairABI,
			Target: poolAddress,
			Method: pairMethodReserve1,
		}, []any{&getReservesResult.Reserve1})
	} else {
		getReservesRequest.AddCall(&ethrpc.Call{
			ABI:    uniswapV2PairABI,
			Target: poolAddress,
			Method: pairMethodGetReserves,
		}, []any{&getReservesResult})
	}

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

	return d.logDecoder.Decode(logs)
}
