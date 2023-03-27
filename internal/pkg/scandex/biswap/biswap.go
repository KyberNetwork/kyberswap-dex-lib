package biswap

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return uniswap.NewWithFunc(scanDexCfg, scanService, uniswap.Option{
		UpdateReserveFunc:          updateReservesFunc,
		UpdateNewPoolFunc:          uniswap.UpdateNewPoolFunc,
		DexType:                    constant.PoolTypes.Uni,
		FactoryAbi:                 abis.BiswapFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	})
}

func updateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var calls = make([]*repository.TryCallParams, 0)

	reserves := make([]uniswap.Reserves, len(pools))
	swapFee := make([]uint32, len(pools))
	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.BiswapPair,
			Target: pool.Address,
			Method: "getReserves",
			Params: nil,
			Output: &reserves[i],
		})
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.BiswapPair,
			Target: pool.Address,
			Method: "swapFee",
			Params: nil,
			Output: &swapFee[i],
		})
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	var ret = 0
	for i, pool := range pools {
		reserve := reserves[i]
		if *calls[2*(i)].Success && *calls[2*(i)+1].Success {
			swapFee := float64(swapFee[i]) / 1000
			scanService.UpdatePoolSwapFee(ctx, pool.Address, swapFee)
			err := scanService.UpdatePoolReserve(ctx, pool.Address, int64(reserve.BlockTimestampLast), []string{
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
