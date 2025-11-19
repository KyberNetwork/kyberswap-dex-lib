package uniswapv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

	reserveData, blockNumber, err := d.getReserves(ctx, p.Address, &params)
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

	fee, err := d.getFee(ctx, p.Address)
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

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, params *pool.GetNewPoolStateParams) (
	ReserveData, *big.Int, error) {
	reserveData, blockNumber, err := d.getReservesFromLogs(params)
	if err != nil || reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string) (uint64, error) {
	feeTracker := d.feeTracker
	if feeTracker == nil {
		return d.config.Fee, nil
	}
	return feeTracker.GetFee(ctx, poolAddress, d.config.FactoryAddress)
}

func (d *PoolTracker) updatePool(p entity.Pool, reserveData ReserveData, fee uint64,
	blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		Fee:          fee,
		FeePrecision: d.config.FeePrecision,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber.Uint64()

	// Keep pool listing timestamp if reserves unchanged since pool creation
	p.Timestamp = max(p.Timestamp, int64(reserveData.BlockTimestampLast))

	return p, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, *big.Int, error) {
	var getReservesResult ReserveData
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []any{&getReservesResult})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}

	return ReserveData{
		Reserve0:           getReservesResult.Reserve0,
		Reserve1:           getReservesResult.Reserve1,
		BlockTimestampLast: getReservesResult.BlockTimestampLast,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) getReservesFromLogs(params *pool.GetNewPoolStateParams) (ReserveData, *big.Int, error) {
	if len(params.Logs) == 0 {
		return ReserveData{}, nil, nil
	}

	return d.logDecoder.Decode(params.Logs, params.BlockHeaders)
}
