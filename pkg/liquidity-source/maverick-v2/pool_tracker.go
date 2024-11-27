package maverickv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	GetStateResult struct {
		ReserveA           *big.Int `json:"reserveA"`
		ReserveB           *big.Int `json:"reserveB"`
		LastTwaD8          int64    `json:"lastTwaD8"`
		LastLogPriceD8     int64    `json:"lastLogPriceD8"`
		LastTimestamp      *big.Int `json:"lastTimestamp"`
		ActiveTick         int32    `json:"activeTick"`
		IsLocked           bool     `json:"isLocked"`
		BinCounter         uint32   `json:"binCounter"`
		ProtocolFeeRatioD3 uint8    `json:"protocolFeeRatioD3"`
	}

	// because the result is a tuple with internal type = struct IMaverickV2Pool.State, we need to wrap it in a struct like this
	GetStateResultWrapper struct {
		GetStateResult
	}
)

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
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	state, blockNumber, err := t.getState(ctx, p.Address)
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

	logger.WithFields(
		logger.Fields{
			"pool_id":          p.Address,
			"old_reserve":      p.Reserves,
			"new_reserve":      state,
			"old_block_number": p.BlockNumber,
			"new_block_number": blockNumber,
			"duration_ms":      time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting new pool state")

	return t.updatePool(p, state, blockNumber)
}

func (t *PoolTracker) getState(ctx context.Context, poolAddress string) (State, *big.Int, error) {
	var getStateResult GetStateResultWrapper

	getStateRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(true)

	getStateRequest.AddCall(&ethrpc.Call{
		ABI:    maverickV2PoolABI,
		Target: poolAddress,
		Method: poolMethodGetState,
		Params: nil,
	}, []interface{}{
		&getStateResult,
	})

	resp, err := getStateRequest.TryBlockAndAggregate()
	if err != nil {
		return State{}, nil, err
	}

	return State{
		ReserveA:      getStateResult.ReserveA,
		ReserveB:      getStateResult.ReserveB,
		LastTimestamp: getStateResult.LastTimestamp.Int64(),
	}, resp.BlockNumber, nil
}

func (t *PoolTracker) updatePool(pool entity.Pool, state State, blockNumber *big.Int) (entity.Pool, error) {
	pool.Reserves = entity.PoolReserves{
		state.ReserveA.String(),
		state.ReserveB.String(),
	}
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = state.LastTimestamp

	return pool, nil
}
