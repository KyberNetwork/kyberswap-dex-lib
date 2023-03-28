package dmm

import (
	"encoding/json"
	"time"

	"context"

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
		DexType:                    constant.PoolTypes.Dmm,
		FactoryAbi:                 abis.DmmFactory,
		FactoryGetPairMethodCall:   "allPools",
		FactoryPairCountMethodCall: "allPoolsLength",
	})
}

func updateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var calls = make([]*repository.TryCallParams, 0)
	reserves := make([]TradeInfo, len(pools))
	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.DmmPool,
			Target: pool.Address,
			Method: "getTradeInfo",
			Params: nil,
			Output: &reserves[i],
		})
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	var ret = 0
	for i, pool := range pools {
		reserve := reserves[i]
		extra := ExtraField{
			VReserves: []string{
				reserve.VReserve0.String(),
				reserve.VReserve1.String(),
			},
			FeeInPrecision: reserve.FeeInPrecision.String(),
		}
		extraBytes, _ := json.Marshal(extra)
		if *calls[i].Success && (pool.Extra != string(extraBytes) ||
			pool.Reserves[0] != reserve.Reserve0.String() || pool.Reserves[1] != reserve.Reserve1.String()) {
			scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
			err := scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), []string{
				reserve.Reserve0.String(),
				reserve.Reserve1.String(),
			})
			if err != nil {
				logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			} else {
				ret++
			}
		}
	}
	return ret
}
