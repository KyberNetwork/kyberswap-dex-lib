package factory

import (
	"encoding/json"
	"math/big"
	"strings"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/pkg/logger"

	curveAave "github.com/KyberNetwork/router-service/internal/pkg/core/curve-aave"
	curveBase "github.com/KyberNetwork/router-service/internal/pkg/core/curve-base"
	curveCompound "github.com/KyberNetwork/router-service/internal/pkg/core/curve-compound"
	curvePlainOracle "github.com/KyberNetwork/router-service/internal/pkg/core/curve-plain-oracle"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"

	"github.com/ethereum/go-ethereum/common"
)

// the get_pool_coins method's response field sizes is fixed = 8
type PoolCoinsFromGetter struct {
	Coins              [8]common.Address
	UnderlyingCoins    [8]common.Address
	Decimals           [8]*big.Int
	UnderlyingDecimals [8]*big.Int
}

type PoolCoins struct {
	Coins              []string
	UnderlyingCoins    []string
	Decimals           []*big.Int
	UnderlyingDecimals []*big.Int
}

func CheckAndFetchPlainAndLendingPools(
	ctx context.Context,
	dex string,
	scanService *service.ScanService,
	mainRegistry string,
	getter string,
	poolAddresses []common.Address,
) error {
	// Get Pool LP Token
	calls := make([]*repository.CallParams, 0, len(poolAddresses))
	lpAddreses := make([]common.Address, len(poolAddresses))

	mainRegistryCallParamsFactory := repository.CallParamsFactory(abis.CurveMainRegistry, mainRegistry)

	for i := 0; i < len(poolAddresses); i++ {
		calls = append(
			calls,
			mainRegistryCallParamsFactory(MainRegistryMethodGetLPToken, &lpAddreses[i], []interface{}{poolAddresses[i]}),
		)
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	// Get Pool Coins (from Getter SC) -> [coins + underlying_coin + decimals + underlying_decimals]
	calls = calls[:0]
	poolCoinsFromGetter := make([]PoolCoinsFromGetter, len(poolAddresses))
	var poolCoins []PoolCoins
	for i := 0; i < len(poolAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveGetter,
			Target: getter,
			Method: PoolGetterMethodGetPoolCoins,
			Params: []interface{}{poolAddresses[i]},
			Output: &poolCoinsFromGetter[i],
		})
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i := 0; i < len(poolCoinsFromGetter); i++ {
		var tmp1 []string
		var tmp2 []string

		for j := range poolCoinsFromGetter[i].Coins {
			if poolCoinsFromGetter[i].Coins[j].Hex() != AddressZero {
				if strings.EqualFold(poolCoinsFromGetter[i].Coins[j].Hex(), AddressEther) {
					tmp1 = append(tmp1, strings.ToLower(constant.WETH9[uint(scanService.Config().ChainID)].Address.Hex()))
				} else {
					tmp1 = append(tmp1, strings.ToLower(poolCoinsFromGetter[i].Coins[j].Hex()))
				}

				if strings.EqualFold(poolCoinsFromGetter[i].UnderlyingCoins[j].Hex(), AddressEther) {
					tmp2 = append(tmp2, strings.ToLower(constant.WETH9[uint(scanService.Config().ChainID)].Address.Hex()))
				} else {
					tmp2 = append(tmp2, strings.ToLower(poolCoinsFromGetter[i].UnderlyingCoins[j].Hex()))
				}
			}
		}

		poolCoins = append(poolCoins, PoolCoins{
			Coins:              tmp1,
			UnderlyingCoins:    tmp2,
			Decimals:           poolCoinsFromGetter[i].Decimals[:],
			UnderlyingDecimals: poolCoinsFromGetter[i].UnderlyingDecimals[:],
		})
	}

	// Get aPrecision
	aPrecisions, err := GetAprecisions(ctx, scanService, poolAddresses)
	if err != nil {
		return err
	}

	// Get Pool Rates
	tryCalls := make([]*repository.TryCallParams, 0, len(poolAddresses))
	// the get_rates method's response size is fixed = 8
	rates := make([][8]*big.Int, len(poolAddresses))

	mainRegistryTryCallParamsFactory := repository.TryCallParamsFactory(abis.CurveMainRegistry, mainRegistry)

	for i := 0; i < len(poolAddresses); i++ {
		tryCalls = append(
			tryCalls,
			mainRegistryTryCallParamsFactory(MainRegistryMethodGetRates, &rates[i], []interface{}{poolAddresses[i]}),
		)
	}
	if err := scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	// Check for valid pools
	validCheck := make([]bool, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		validCheck[i] = true

		if rates[i][0] == nil {
			validCheck[i] = false
		}
	}

	tryCalls = tryCalls[:0]
	aaveSignatures := make([]*big.Int, len(poolAddresses))
	oracles := make([]common.Address, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		tryCalls = append(
			tryCalls,
			&repository.TryCallParams{
				ABI:    abis.CurveAave,
				Target: poolAddresses[i].Hex(),
				Method: AavePoolMethodOffpegFeeMultiplier,
				Params: nil,
				Output: &aaveSignatures[i],
			},
			&repository.TryCallParams{
				ABI:    abis.CurvePlainOraclePool,
				Target: poolAddresses[i].Hex(),
				Method: PlainOraclePoolMethodOracle,
				Params: nil,
				Output: &oracles[i],
			},
		)
	}

	if err := scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	poolTypes := make([]string, 0, len(poolAddresses))

	for i := range poolAddresses {
		if !eth.IsZeroAddress(oracles[i]) {
			poolTypes = append(poolTypes, constant.PoolTypes.CurvePlainOracle)
		} else if IsPlainPool(poolCoins[i].Coins, poolCoins[i].UnderlyingCoins) {
			poolTypes = append(poolTypes, constant.PoolTypes.CurveBase)
		} else if aaveSignatures[i] != nil {
			poolTypes = append(poolTypes, constant.PoolTypes.CurveAave)
		} else {
			poolTypes = append(poolTypes, constant.PoolTypes.CurveLending)
		}
	}

	for i := range poolAddresses {
		if scanService.ExistPool(ctx, strings.ToLower(poolAddresses[i].Hex())) || !validCheck[i] {
			continue
		}

		var tokens []*entity.PoolToken
		reserves := make(entity.PoolReserves, 0, len(poolCoins[i].Coins)+1)
		var staticExtraBytes []byte
		if poolTypes[i] == constant.PoolTypes.CurveBase {
			var staticExtra = curveBase.PoolStaticExtra{
				LpToken:    strings.ToLower(lpAddreses[i].Hex()),
				APrecision: aPrecisions[i].String(),
			}
			for j := range poolCoins[i].Coins {
				precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), poolCoins[i].Decimals[j]), nil)
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
				staticExtra.Rates = append(staticExtra.Rates, new(big.Int).Mul(precision, rates[i][j]).String())

				_, err := scanService.FetchOrGetToken(ctx, poolCoins[i].Coins[j])
				if err != nil {
					return err
				}

				tokens = append(
					tokens, &entity.PoolToken{
						Address:   strings.ToLower(poolCoins[i].Coins[j]),
						Weight:    1,
						Swappable: true,
					},
				)
				reserves = append(reserves, ReserveZero)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)
		} else if poolTypes[i] == constant.PoolTypes.CurvePlainOracle {
			var staticExtra = curvePlainOracle.PoolStaticExtra{
				LpToken:    strings.ToLower(lpAddreses[i].Hex()),
				APrecision: aPrecisions[i].String(),
				Oracle:     oracles[i].Hex(),
			}
			for j := range poolCoins[i].Coins {
				precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), poolCoins[i].Decimals[j]), nil)
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())

				_, err := scanService.FetchOrGetToken(ctx, poolCoins[i].Coins[j])
				if err != nil {
					return err
				}

				tokens = append(
					tokens, &entity.PoolToken{
						Address:   strings.ToLower(poolCoins[i].Coins[j]),
						Weight:    1,
						Swappable: true,
					},
				)
				reserves = append(reserves, ReserveZero)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)
		} else if poolTypes[i] == constant.PoolTypes.CurveAave {
			var staticExtra = curveAave.PoolStaticExtra{
				LpToken:          strings.ToLower(lpAddreses[i].Hex()),
				UnderlyingTokens: poolCoins[i].UnderlyingCoins,
			}
			for j := range poolCoins[i].Coins {
				precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), poolCoins[i].UnderlyingDecimals[j]), nil)
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
				if poolCoins[i].UnderlyingCoins[j] != AddressZero && poolCoins[i].UnderlyingCoins[j] != AddressEther {
					_, err := scanService.FetchOrGetTokenType(
						ctx,
						poolCoins[i].Coins[j],
						"aave",
						poolCoins[i].UnderlyingCoins[j],
					)
					if err != nil {
						return err
					}
					tokens = append(
						tokens, &entity.PoolToken{
							Address:   strings.ToLower(poolCoins[i].Coins[j]),
							Weight:    1,
							Swappable: false,
						},
					)
				}
				reserves = append(reserves, ReserveZero)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)
		} else if poolTypes[i] == constant.PoolTypes.CurveLending {
			var staticExtra = curveCompound.PoolStaticExtra{
				LpToken:          strings.ToLower(lpAddreses[i].Hex()),
				UnderlyingTokens: poolCoins[i].UnderlyingCoins,
			}

			for j := range poolCoins[i].Coins {
				precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), poolCoins[i].UnderlyingDecimals[j]), nil)
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
				// staticExtra.Rates = append(staticExtra.Rates, new(big.Int).Mul(precision, rates[i][j]).String())
				if poolCoins[i].UnderlyingCoins[j] != AddressZero && poolCoins[i].UnderlyingCoins[j] != AddressEther {
					token, err := scanService.FetchOrGetTokenType(
						ctx,
						poolCoins[i].Coins[j],
						"aave",
						poolCoins[i].UnderlyingCoins[j],
					)
					if err != nil {
						return err
					}

					if poolTypes[i] == constant.PoolTypes.CurveLending && strings.Contains(strings.ToLower(token.Name), "compound") {
						poolTypes[i] = constant.PoolTypes.CurveCompound
					}
					tokens = append(
						tokens, &entity.PoolToken{
							Address:   strings.ToLower(poolCoins[i].Coins[j]),
							Weight:    1,
							Swappable: false,
						},
					)
				}
				reserves = append(reserves, ReserveZero)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)
		}

		// This is for the totalSupply - the last item in slice
		reserves = append(reserves, ReserveZero)

		var newPool = entity.Pool{
			Address:     strings.ToLower(poolAddresses[i].Hex()),
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    dex,
			Type:        poolTypes[i],
			Timestamp:   0,
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}

		if err := scanService.SavePool(ctx, newPool); err != nil {
			return err
		}
	}

	return nil
}
