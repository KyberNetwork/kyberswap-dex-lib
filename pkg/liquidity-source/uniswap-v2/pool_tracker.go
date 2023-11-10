package uniswapv2

import (
	"context"
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
		Decode(logs []types.Log) (ReserveData, error)
	}

	PoolTracker struct {
		ethrpcClient *ethrpc.Client
		logDecoder   ILogDecoder
	}

	GetReservesResult struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}
)

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
		logDecoder:   NewLogDecoder(),
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

	reserveData, err := d.getReserves(ctx, p.Address, params.Logs)
	if err != nil {
		return p, err
	}

	p = d.updatePool(p, reserveData)

	return p, nil
}

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, error) {
	reserveData, err := d.getReservesFromLogs(logs)
	if err != nil {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	if reserveData.IsZero() {
		return d.getReservesFromRPCNode(ctx, poolAddress)
	}

	return reserveData, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData) entity.Pool {
	if pool.BlockNumber > reserveData.BlockNumber {
		return pool
	}

	pool.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	pool.BlockNumber = reserveData.BlockNumber
	pool.Timestamp = time.Now().Unix()

	return pool
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, error) {
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
		return ReserveData{}, err
	}

	return ReserveData{
		Reserve0:    getReservesResult.Reserve0,
		Reserve1:    getReservesResult.Reserve1,
		BlockNumber: resp.BlockNumber.Uint64(),
	}, nil
}

func (d *PoolTracker) getReservesFromLogs(logs []types.Log) (ReserveData, error) {
	if len(logs) == 0 {
		return ReserveData{}, nil
	}

	return d.logDecoder.Decode(logs)
}
