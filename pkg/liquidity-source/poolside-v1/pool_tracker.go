package poolsidev1

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_address": p.Address}).Info("Started getting new pool state")

	reserveData, blockNumber, err := d.getReservesFromRPCNode(ctx, p.Address)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_address":      p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	rebaseTokenInfoMap := extra.RebaseTokenInfoMap

	if err := d.updateRebaseTokenInfoMap(ctx, &rebaseTokenInfoMap); err != nil {
		return p, err
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_address":     p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      reserveData,
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, reserveData, blockNumber, rebaseTokenInfoMap)
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, poolAddress string) (ReserveData, *big.Int, error) {
	var getReservesResult GetReservesResult

	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReservesRequest.AddCall(&ethrpc.Call{
		ABI:    poolsideV1PairABI,
		Target: poolAddress,
		Method: pairMethodGetLiquidityBalances,
		Params: nil,
	}, []interface{}{&getReservesResult.Pool0, &getReservesResult.Pool1, &getReservesResult.Reservoir0, &getReservesResult.Reservoir1, &getReservesResult.BlockTimestampLast})

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}

	return ReserveData{
		Pool0: getReservesResult.Pool0,
		Pool1: getReservesResult.Pool1,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) callUnderlyingToWrapper(ctx context.Context, tokenAddress string, uAmount *big.Int) (*big.Int, error) {
	var result *big.Int

	callRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	callRequest.AddCall(&ethrpc.Call{
		ABI:    poolsideV1ButtonTokenABI,
		Target: tokenAddress,
		Method: buttonTokenMethodUnderlyingToWrapper,
		Params: []interface{}{uAmount},
	}, []interface{}{&result})

	if _, err := callRequest.Call(); err != nil {
		return nil, err
	}

	return result, nil
}

func (d *PoolTracker) callWrapperToUnderlying(ctx context.Context, tokenAddress string, amount *big.Int) (*big.Int, error) {
	var result *big.Int

	callRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	callRequest.AddCall(&ethrpc.Call{
		ABI:    poolsideV1ButtonTokenABI,
		Target: tokenAddress,
		Method: buttonTokenMethodWrapperToUnderlying,
		Params: []interface{}{amount},
	}, []interface{}{&result})

	if _, err := callRequest.Call(); err != nil {
		return nil, err
	}

	return result, nil
}

func (d *PoolTracker) updateRebaseTokenInfoMap(ctx context.Context, rebaseTokenInfoMap *map[string]RebaseTokenInfo) error {
	for tokenAddress, info := range *rebaseTokenInfoMap {
		if strings.TrimSpace(info.UnderlyingToken) == "" {
			continue
		}

		amount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(info.Decimals)), nil)

		underlyingToWrapper, err := d.callUnderlyingToWrapper(ctx, tokenAddress, amount)
		if err != nil {
			return err
		}

		wrapperToUnderlying, err := d.callWrapperToUnderlying(ctx, tokenAddress, amount)
		if err != nil {
			return err
		}

		info.WrapRatio = underlyingToWrapper
		info.UnwrapRatio = wrapperToUnderlying

		(*rebaseTokenInfoMap)[tokenAddress] = info
	}

	return nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData, blockNumber *big.Int, rebaseTokenInfoMap map[string]RebaseTokenInfo) (entity.Pool, error) {
	extra := Extra{
		Fee:                d.config.Fee,
		FeePrecision:       d.config.FeePrecision,
		RebaseTokenInfoMap: rebaseTokenInfoMap,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{
		reserveData.Pool0.String(),
		reserveData.Pool1.String(),
	}

	pool.Extra = string(extraBytes)
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}
