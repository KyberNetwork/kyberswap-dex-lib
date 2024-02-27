package bancor_v21

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
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	reserveData, blockNumber, err := d.getReserves(ctx, p, params.Logs)
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

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData []*big.Int, fee uint64, blockNumber *big.Int) (entity.Pool, error) {
	extra := ExtraInner{
		conversionFee: fee,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{}
	for _, reserve := range reserveData {
		pool.Reserves = append(pool.Reserves, reserve.String())
	}

	pool.Extra = string(extraBytes)
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getReserves(ctx context.Context, p entity.Pool, logs []types.Log) ([]*big.Int, *big.Int, error) {
	return d.getReservesFromRPCNode(ctx, p.Address, p.Tokens)
}

func (d *PoolTracker) getFee(ctx context.Context, poolAddress string) (uint64, error) {
	fee := big.NewInt(0)
	if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    converterABI,
		Target: poolAddress,
		Method: converterGetFee,
		Params: nil,
	}, []interface{}{&fee}).Call(); err != nil {
		return 0, err
	}
	return fee.Uint64(), nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string, tokens []*entity.PoolToken) ([]*big.Int, *big.Int, error) {
	reserves := make([]*big.Int, len(tokens))
	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		getReservesRequest.AddCall(&ethrpc.Call{
			ABI:    converterABI,
			Target: poolAddress,
			Method: converterGetReserve,
			Params: []interface{}{token},
		}, []interface{}{&reserves[i]})
	}

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return reserves, resp.BlockNumber, nil
}
