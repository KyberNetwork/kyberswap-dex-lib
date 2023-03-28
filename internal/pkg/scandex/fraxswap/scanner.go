package fraxswap

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return uniswap.NewWithFunc(scanDexCfg, scanService, uniswap.Option{
		UpdateReserveFunc:          updateReservesFunc,
		UpdateNewPoolFunc:          uniswap.UpdateNewPoolFunc,
		DexType:                    constant.PoolTypes.Fraxswap,
		FactoryAbi:                 abis.FraxswapFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	})
}

func updateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	calls := make([]*repository.TryCallParams, 0, len(pools))
	getReserveAfterTwammOutputs := make([]GetReserveAfterTwammOutput, len(pools))
	feeOutputs := make([]FeeOutput, len(pools))

	for i, pool := range pools {
		getReserveAfterTwammCall := &repository.TryCallParams{
			ABI:    abis.FraxswapPair,
			Target: pool.Address,
			Method: "getReserveAfterTwamm",
			Params: []interface{}{big.NewInt(time.Now().Unix())},
			Output: &getReserveAfterTwammOutputs[i],
		}

		feeCall := &repository.TryCallParams{
			ABI:    abis.FraxswapPair,
			Target: pool.Address,
			Method: "fee",
			Params: nil,
			Output: &feeOutputs[i],
		}

		calls = append(calls, getReserveAfterTwammCall)
		calls = append(calls, feeCall)
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}

	ret := 0
	for i, pool := range pools {
		if !*calls[i].Success {
			continue
		}

		getReserveAfterTwammOutput := getReserveAfterTwammOutputs[i]
		feeOutput := feeOutputs[i]

		extra := Extra{
			Reserve0: getReserveAfterTwammOutput.Reserve0,
			Reserve1: getReserveAfterTwammOutput.Reserve1,
			Fee:      feeOutput.Fee,
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"pool":  pool.Address,
				"error": err,
			}).Error("json marshal failed")

			continue
		}

		if pool.Extra != string(extraBytes) {
			err = scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
			if err != nil {
				logger.WithFields(map[string]interface{}{
					"pool":  pool.Address,
					"error": err,
				}).Error("failed to update pool extra")

				continue
			}
		}

		if pool.Reserves[0] != getReserveAfterTwammOutput.Reserve0.String() || pool.Reserves[1] != getReserveAfterTwammOutput.Reserve1.String() {
			err = scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), []string{
				getReserveAfterTwammOutput.Reserve0.String(),
				getReserveAfterTwammOutput.Reserve1.String(),
			})
			if err != nil {
				logger.WithFields(map[string]interface{}{
					"pool":  pool.Address,
					"error": err,
				}).Error("failed to update pool reserve")

				continue
			}
		}

		ret++
	}
	return ret
}
