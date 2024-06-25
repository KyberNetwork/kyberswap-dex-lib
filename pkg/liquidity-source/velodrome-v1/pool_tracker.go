package velodromev1

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type (
	ILogDecoder interface {
		Decode(logs []types.Log) (ReserveData, uint64, error)
	}

	PoolTracker struct {
		config             *Config
		ethrpcClient       *ethrpc.Client
		logDecoder         ILogDecoder
		feeTrackerRegistry map[string]IFeeTracker
	}
)

var (
	ErrFeeTrackerMissing = errors.New("fee tracker missing")
)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		//logDecoder:   NewLogDecoder(),
		feeTrackerRegistry: map[string]IFeeTracker{
			FeeTrackerIDVelodrome: &VelodromeFeeTracker{ethrpcClient: ethrpcClient},
			FeeTrackerIDStratum:   &StratumFeeTracker{ethrpcClient: ethrpcClient},
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

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, uint64, error) {
	reserveData, blockNumber, err := d.getReservesFromLogs(logs)
	if err != nil {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	if reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getFactoryData(ctx context.Context, blockNumber uint64) (bool, error) {
	var (
		isPaused bool
	)

	getFactoryDataRequest := d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(big.NewInt(int64(blockNumber)))

	getFactoryDataRequest.AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: d.config.FactoryAddress,
		Method: pairFactoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&isPaused})

	if _, err := getFactoryDataRequest.TryBlockAndAggregate(); err != nil {
		return false, err
	}

	return isPaused, nil
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string, isStable bool, blockNumber uint64) (uint64, error) {
	feeTracker, ok := d.feeTrackerRegistry[d.config.FeeTracker]
	if !ok {
		return 0, ErrFeeTrackerMissing
	}

	blockNumberBI := new(big.Int).SetUint64(blockNumber)

	return feeTracker.GetFee(ctx, poolAddress, isStable, d.config.FactoryAddress, blockNumberBI)
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

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, uint64, error) {
	var (
		getReservesResult GetReservesResult
	)

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, 0, err
	}

	return ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}, resp.BlockNumber.Uint64(), nil
}

func (d *PoolTracker) getReservesFromLogs(logs []types.Log) (ReserveData, uint64, error) {
	if len(logs) == 0 {
		return ReserveData{}, 0, nil
	}

	if d.logDecoder == nil {
		return ReserveData{}, 0, nil
	}

	return d.logDecoder.Decode(logs)
}
