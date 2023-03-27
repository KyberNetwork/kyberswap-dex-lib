package factory

import (
	"encoding/json"
	"math/big"
	"strings"

	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"

	curveMeta "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/curve-meta"

	"github.com/ethereum/go-ethereum/common"
)

func CheckAndFetchMetaPools(
	ctx context.Context,
	dex string,
	scanService *service.ScanService,
	metaPoolsFactory string,
	poolAddresses []common.Address,
) error {
	calls := make([]*repository.CallParams, 0, len(poolAddresses))
	isMeta := make([]bool, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMetaFactory,
			Target: metaPoolsFactory,
			Method: "is_meta",
			Params: []interface{}{poolAddresses[i]},
			Output: &isMeta[i],
		})
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	var poolMetaAddresses []common.Address
	for i := 0; i < len(poolAddresses); i++ {
		if isMeta[i] {
			poolMetaAddresses = append(poolMetaAddresses, poolAddresses[i])
		}
	}

	aPrecisions, err := GetAprecisions(ctx, scanService, poolMetaAddresses)
	if err != nil {
		return err
	}

	calls = calls[:0]
	basePoolsAddresses := make([]common.Address, len(poolMetaAddresses))
	for i := 0; i < len(poolMetaAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMetaFactory,
			Target: metaPoolsFactory,
			Method: "get_base_pool",
			Params: []interface{}{poolMetaAddresses[i]},
			Output: &basePoolsAddresses[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	calls = calls[:0]
	// the get_coins method's response size is fixed = 4
	var poolCoinsAddresses = make([][4]common.Address, len(poolMetaAddresses))
	for i := 0; i < len(poolMetaAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMetaFactory,
			Target: metaPoolsFactory,
			Method: "get_coins",
			Params: []interface{}{poolMetaAddresses[i]},
			Output: &poolCoinsAddresses[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	calls = calls[:0]
	var poolCoinsDecimals = make([][4]*big.Int, len(poolMetaAddresses))
	for i := 0; i < len(poolMetaAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMetaFactory,
			Target: metaPoolsFactory,
			Method: "get_decimals",
			Params: []interface{}{poolMetaAddresses[i]},
			Output: &poolCoinsDecimals[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	calls = calls[:0]
	var underlyingCoinsAddresses = make([][8]common.Address, len(poolMetaAddresses))
	for i := 0; i < len(poolMetaAddresses); i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.CurveMetaFactory,
			Target: metaPoolsFactory,
			Method: "get_underlying_coins",
			Params: []interface{}{poolMetaAddresses[i]},
			Output: &underlyingCoinsAddresses[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i := range poolMetaAddresses {
		if scanService.ExistPool(ctx, strings.ToLower(poolMetaAddresses[i].Hex())) {
			continue
		}

		var tokens []*entity.PoolToken
		reserves := make(entity.PoolReserves, 0, len(poolCoinsAddresses[i])+1)
		var staticExtraBytes []byte
		var staticExtra = curveMeta.PoolStaticExtra{
			LpToken:          strings.ToLower(poolMetaAddresses[i].Hex()),
			BasePool:         strings.ToLower(basePoolsAddresses[i].Hex()),
			RateMultiplier:   new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(36), poolCoinsDecimals[i][0]), nil).String(), // 36 - coin0-decimal
			APrecision:       aPrecisions[i].String(),
			UnderlyingTokens: CommonAddressesToStrings(underlyingCoinsAddresses[i][:]),
		}

		for j := range poolCoinsAddresses[i] {
			if poolCoinsAddresses[i][j].Hex() != AddressZero {
				precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), poolCoinsDecimals[i][j]), nil)
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
				staticExtra.Rates = append(staticExtra.Rates, "")
			}
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)

		for j := range poolCoinsAddresses[i] {
			if poolCoinsAddresses[i][j].Hex() != AddressZero {
				if _, err := scanService.FetchOrGetToken(ctx, poolCoinsAddresses[i][j].Hex()); err != nil {
					return err
				}
				tokens = append(
					tokens, &entity.PoolToken{
						Address:   strings.ToLower(poolCoinsAddresses[i][j].Hex()),
						Weight:    1,
						Swappable: true,
					},
				)
				reserves = append(reserves, ReserveZero)
			}
		}

		// This is for the totalSupply - the last item in slice
		reserves = append(reserves, ReserveZero)

		var newPool = entity.Pool{
			Address:     strings.ToLower(poolMetaAddresses[i].Hex()),
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    dex,
			Type:        constant.PoolTypes.CurveMeta,
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
