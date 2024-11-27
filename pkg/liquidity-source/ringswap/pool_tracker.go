package ringswap

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   uniswapv2.ILogDecoder
	}
)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   uniswapv2.NewLogDecoder(),
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	fwReserves, originalReserves, blockNumber, err := d.getReserves(ctx, p.Address, p.Tokens)
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
				"new_reserve":      fwReserves,
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, fwReserves, originalReserves, blockNumber)
}

func (d *PoolTracker) getReserves(ctx context.Context, poolAddress string, tokens []*entity.PoolToken) (uniswapv2.ReserveData,
	uniswapv2.ReserveData, *big.Int, error) {
	if len(tokens) < 4 {
		return uniswapv2.ReserveData{}, uniswapv2.ReserveData{}, nil, errors.New("invalid number of tokens")
	}

	var (
		getReservesResult uniswapv2.GetReservesResult

		originalReserve0 = bignumber.ZeroBI
		originalReserve1 = bignumber.ZeroBI

		originalToken0, fwToken0 = tokens[0], tokens[2]
		originalToken1, fwToken1 = tokens[1], tokens[3]
	)

	if (originalToken0.Address == fwToken0.Address) || (originalToken1.Address == fwToken1.Address) {
		return uniswapv2.ReserveData{}, uniswapv2.ReserveData{}, nil, errors.New("waiting for fetching origin token address")
	}

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})
	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: originalToken0.Address,
		Method: pairMethodBalanceOf,
		Params: []interface{}{common.HexToAddress(fwToken0.Address)},
	}, []interface{}{&originalReserve0})
	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: originalToken1.Address,
		Method: pairMethodBalanceOf,
		Params: []interface{}{common.HexToAddress(fwToken1.Address)},
	}, []interface{}{&originalReserve1})

	resp, err := getReservesRequest.Aggregate()
	if err != nil {
		return uniswapv2.ReserveData{}, uniswapv2.ReserveData{}, nil, err
	}

	fwReserves := uniswapv2.ReserveData{
		Reserve0: getReservesResult.Reserve0,
		Reserve1: getReservesResult.Reserve1,
	}

	originalReserves := uniswapv2.ReserveData{
		Reserve0: originalReserve0,
		Reserve1: originalReserve1,
	}

	return fwReserves, originalReserves, resp.BlockNumber, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, fwReserves, originalReserves uniswapv2.ReserveData, blockNumber *big.Int) (entity.Pool, error) {
	extra, err := json.Marshal(&originalReserves)
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Reserves = entity.PoolReserves{
		fwReserves.Reserve0.String(),
		fwReserves.Reserve1.String(),
		"1",
		"1",
	}

	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extra)

	return pool, nil
}
