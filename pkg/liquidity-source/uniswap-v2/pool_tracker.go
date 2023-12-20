package uniswapv2

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type (
	ILogDecoder interface {
		Decode(logs []types.Log) (ReserveData, *big.Int, error)
	}

	PoolTracker struct {
		config             *Config
		ethrpcClient       *ethrpc.Client
		logDecoder         ILogDecoder
		feeTrackerRegistry map[string]IFeeTracker
	}

	GetReservesResult struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}
)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   NewLogDecoder(),
		feeTrackerRegistry: map[string]IFeeTracker{
			FeeTrackerIDMMF:       &MMFFeeTracker{ethrpcClient: ethrpcClient},
			FeeTrackerIDMdex:      &MDexFeeTracker{ethrpcClient: ethrpcClient},
			FeeTrackerIDShibaswap: &ShibaswapFeeTracker{ethrpcClient: ethrpcClient},
			FeeTrackerIDDefiswap:  &DefiSwapFeeTracker{ethrpcClient: ethrpcClient},
		},
	}, nil
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

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, *big.Int, error) {
	reserveData, blockNumber, err := d.getReservesFromLogs(logs)
	if err != nil {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	if reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string, blockNumber *big.Int) (uint64, error) {
	feeTracker, ok := d.feeTrackerRegistry[d.config.FeeTracker]
	if !ok {
		return d.config.Fee, nil
	}

	return feeTracker.GetFee(ctx, poolAddress, d.config.FactoryAddress, blockNumber)
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData, fee uint64, blockNumber *big.Int) (entity.Pool, error) {
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
	var getReservesResult GetReservesResult

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

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
