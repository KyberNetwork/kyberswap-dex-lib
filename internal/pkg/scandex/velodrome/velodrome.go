package velodrome

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

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
	properties, err := uniswap.NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	option := uniswap.Option{
		UpdateReserveFunc:          updateReservesFunc(properties.FactoryAddress),
		UpdateNewPoolFunc:          UpdateNewPoolFunc,
		DexType:                    constant.PoolTypes.Velodrome,
		FactoryAbi:                 abis.VelodromeFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	}

	return uniswap.NewWithFunc(scanDexCfg, scanService, option)
}

func UpdateNewPoolFunc(
	ctx context.Context,
	scanService *service.ScanService,
	option uniswap.Option,
	scanDexCfg *config.ScanDex,
	properties interface{},
	pairAddresses []common.Address,
) error {
	var calls = make([]*repository.CallParams, 0)
	var limit = len(pairAddresses)
	calls = make([]*repository.CallParams, 0)
	var pairMetadata = make([]metadata, limit)

	for i := 0; i < limit; i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.VelodromePair,
			Target: pairAddresses[i].Hex(),
			Method: "metadata",
			Params: nil,
			Output: &pairMetadata[i],
		})
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i, pair := range pairAddresses {
		p := strings.ToLower(pair.Hex())
		token0Address := strings.ToLower(pairMetadata[i].T0.String())
		token1Address := strings.ToLower(pairMetadata[i].T1.String())
		if scanService.ExistPool(ctx, p) {
			continue
		}

		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    50,
			Decimals:  uint8(len(pairMetadata[i].Dec0.String()) - 1),
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    50,
			Decimals:  uint8(len(pairMetadata[i].Dec1.String()) - 1),
			Swappable: true,
		}
		if _, err := scanService.FetchOrGetToken(ctx, token0.Address); err != nil {
			return err
		}
		if _, err := scanService.FetchOrGetToken(ctx, token1.Address); err != nil {
			return err
		}

		sExtra := StaticExtra{
			Stable: pairMetadata[i].St,
		}
		staticExtra, err := json.Marshal(sExtra)
		if err != nil {
			return err
		}

		var pool = entity.Pool{
			Address:     p,
			ReserveUsd:  0,
			SwapFee:     properties.(uniswap.Properties).SwapFee,
			Exchange:    scanDexCfg.Id,
			Type:        option.DexType,
			Timestamp:   0,
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtra),
		}
		err = scanService.SavePool(ctx, pool)
		if err != nil {
			logger.Errorf("can not save pool address=%v err=%v", pool.Address, err)
			return err
		}
	}
	return nil
}

func updateReservesFunc(factoryAddress string) uniswap.UpdateReserveHandler {
	handler := func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
		var calls = make([]*repository.TryCallParams, 0)
		var reserves = make([]Reserves, len(pools))
		var stableFee *big.Int
		var volatileFee *big.Int

		for i, pool := range pools {
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.VelodromePair,
				Target: pool.Address,
				Method: "getReserves",
				Params: nil,
				Output: &reserves[i],
			})
		}

		// get two types of fee
		{
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.VelodromeFactory,
				Target: factoryAddress,
				Method: "stableFee",
				Params: nil,
				Output: &stableFee,
			})
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.VelodromeFactory,
				Target: factoryAddress,
				Method: "volatileFee",
				Params: nil,
				Output: &volatileFee,
			})
		}

		if err := scanService.TryAggregate(ctx, false, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			return 0
		}

		if !validateGetFeeSuccess(calls[len(calls)-2], calls[len(calls)-1]) {
			logger.Errorf("Velodrome failed to get fee of address %s", calls[len(calls)-1].Target)
			return 0
		}

		var ret = 0
		for i, pool := range pools {
			if !*calls[i].Success {
				logger.Errorf("failed to get reserve: %v", pool.Address)
				continue
			}

			reserve := reserves[i]
			fee := stableFee.Int64()

			extra, err := extractStaticExtra(pool)
			if err != nil {
				logger.Errorf("Velodrome extract static extra err: %v", err)
				continue
			}
			if !extra.Stable {
				fee = volatileFee.Int64()
			}

			err = scanService.UpdatePoolReserve(ctx, pool.Address, reserve.BlockTimestampLast.Int64(), []string{
				reserve.Reserve0.String(),
				reserve.Reserve1.String(),
			})
			if err != nil {
				logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
				continue
			}

			err = scanService.UpdatePoolSwapFee(ctx, pool.Address, float64(fee)/bps)
			if err != nil {
				logger.Errorf("failed to save pool fee: %v err %v", pool.Address, err)
				continue
			}

			ret++
		}

		return ret
	}

	return handler
}

func extractStaticExtra(pool entity.Pool) (StaticExtra, error) {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
	if err != nil {
		return StaticExtra{}, err
	}

	return staticExtra, nil
}

func validateGetFeeSuccess(callStableFee, callVolatileFee *repository.TryCallParams) bool {
	return *callStableFee.Success && *callVolatileFee.Success
}
