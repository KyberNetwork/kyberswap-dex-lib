package camelot

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

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

	return uniswap.NewWithFunc(scanDexCfg, scanService, uniswap.Option{
		UpdateReserveFunc:          updateReservesFunc(properties.FactoryAddress),
		UpdateNewPoolFunc:          updateNewPoolFunc,
		DexType:                    constant.PoolTypes.Camelot,
		FactoryAbi:                 abis.CamelotFactory,
		FactoryGetPairMethodCall:   FactoryMethodAllPairs,
		FactoryPairCountMethodCall: FactoryMethodAllPairsLength,
	})
}

func updateNewPoolFunc(
	ctx context.Context,
	scanService *service.ScanService,
	option uniswap.Option,
	scanDexCfg *config.ScanDex,
	properties interface{},
	pairAddresses []common.Address,
) error {
	var calls []*repository.CallParams
	token0Addresses := make([]common.Address, len(pairAddresses))
	token1Addresses := make([]common.Address, len(pairAddresses))
	feeDenominators := make([]*big.Int, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		callParamsFactory := repository.CallParamsFactory(abis.CamelotPair, pairAddress.Hex())

		calls = append(
			calls,
			callParamsFactory(PairMethodToken0, &token0Addresses[i], nil),
			callParamsFactory(PairMethodToken1, &token1Addresses[i], nil),
			callParamsFactory(PairMethodFeeDenominator, &feeDenominators[i], nil),
		)
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i, pair := range pairAddresses {
		pairAddress := strings.ToLower(pair.Hex())
		token0Address := strings.ToLower(token0Addresses[i].Hex())
		token1Address := strings.ToLower(token1Addresses[i].Hex())
		feeDenominator := feeDenominators[i]

		if scanService.ExistPool(ctx, pairAddress) {
			continue
		}

		token0 := entity.PoolToken{
			Address:   token0Address,
			Weight:    50,
			Swappable: true,
		}

		if _, err := scanService.FetchOrGetToken(ctx, token0.Address); err != nil {
			return err
		}

		token1 := entity.PoolToken{
			Address:   token1Address,
			Weight:    50,
			Swappable: true,
		}

		if _, err := scanService.FetchOrGetToken(ctx, token1.Address); err != nil {
			return err
		}

		staticExtra := StaticExtra{
			FeeDenominator: feeDenominator,
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return err
		}

		pool := entity.Pool{
			Address:     pairAddress,
			ReserveUsd:  0,
			Exchange:    scanDexCfg.Id,
			Type:        option.DexType,
			Timestamp:   0,
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtraBytes),
		}

		if err = scanService.SavePool(ctx, pool); err != nil {
			logger.Errorf("can not save pool address=%v err=%v", pool.Address, err)
			return err
		}
	}
	return nil
}

func updateReservesFunc(factoryAddress string) func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	return func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
		var factory Factory
		factoryCalls := []*repository.CallParams{
			{
				ABI:    abis.CamelotFactory,
				Target: factoryAddress,
				Method: FactoryMethodFeeTo,
				Output: &factory.FeeTo,
			},
			{
				ABI:    abis.CamelotFactory,
				Target: factoryAddress,
				Method: FactoryMethodOwnerFeeShare,
				Output: &factory.OwnerFeeShare,
			},
		}
		if err := scanService.MultiCall(ctx, factoryCalls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			return 0
		}

		var calls []*repository.TryCallParams
		pairs := make([]Pair, len(pools))

		for i, pool := range pools {
			tryCallParamsFactory := repository.TryCallParamsFactory(abis.CamelotPair, pool.Address)

			calls = append(
				calls,
				tryCallParamsFactory(PairMethodStableSwap, &pairs[i].StableSwap, nil),
				tryCallParamsFactory(PairMethodToken0FeePercent, &pairs[i].Token0FeePercent, nil),
				tryCallParamsFactory(PairMethodToken1FeePercent, &pairs[i].Token1FeePercent, nil),
				tryCallParamsFactory(PairMethodPrecisionMultiplier0, &pairs[i].PrecisionMultiplier0, nil),
				tryCallParamsFactory(PairMethodPrecisionMultiplier1, &pairs[i].PrecisionMultiplier1, nil),
				tryCallParamsFactory(PairMethodGetReserves, &pairs[i], nil),
			)
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

			pair := pairs[i]

			extra := Extra{
				StableSwap:           pair.StableSwap,
				Token0FeePercent:     big.NewInt(int64(pair.Token0FeePercent)),
				Token1FeePercent:     big.NewInt(int64(pair.Token1FeePercent)),
				PrecisionMultiplier0: pair.PrecisionMultiplier0,
				PrecisionMultiplier1: pair.PrecisionMultiplier1,
				Factory:              &factory,
			}

			extraBytes, err := json.Marshal(extra)
			if err != nil {
				logger.WithFields(logger.Fields{
					"pool":  pool.Address,
					"error": err,
				}).Error("json marshal failed")

				continue
			}

			if pool.Extra != string(extraBytes) {
				err = scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
				if err != nil {
					logger.WithFields(logger.Fields{
						"pool":  pool.Address,
						"error": err,
					}).Error("failed to update pool extra")

					continue
				}
			}

			if pool.Reserves[0] != pair.Reserve0.String() || pool.Reserves[1] != pair.Reserve1.String() {
				err = scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), []string{
					pair.Reserve0.String(),
					pair.Reserve1.String(),
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
}
