package ethervista

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type (
	ILogDecoder interface {
		Decode(logs []types.Log) (ReserveData, *big.Int, error)
	}

	PoolTracker struct {
		config       *Config
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
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
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

	reserveData, routerAddress, blockNumber, err := d.getRPCState(ctx, p.Address)
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

	return d.updatePool(p, reserveData, routerAddress, blockNumber)
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData, routerAddress common.Address, blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		RouterAddress: routerAddress.Hex(),
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

func (d *PoolTracker) getRPCState(ctx context.Context, poolAddress string) (ReserveData, common.Address, *big.Int, error) {
	var (
		getReservesResult GetReservesResult
		routerAddress     common.Address
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodRouter,
		Params: nil,
	}, []interface{}{&routerAddress})

	resp, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, common.Address{}, nil, err
	}

	return ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}, routerAddress, resp.BlockNumber, nil
}
