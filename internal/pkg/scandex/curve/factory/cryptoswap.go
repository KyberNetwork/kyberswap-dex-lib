package factory

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type ICurve interface {
	ExtractStaticExtra(
		ctx context.Context, scanService *service.ScanService, poolItem PoolItem,
	) (staticExtraBytes []byte)
	ExtractReservesAndTokens(
		ctx context.Context, scanService *service.ScanService, poolItem PoolItem,
	) (reserves entity.PoolReserves, tokens []*entity.PoolToken, err error)
}

type CryptoSwap struct {
	curve ICurve
}

func New(curve ICurve) *CryptoSwap {
	return &CryptoSwap{
		curve: curve,
	}
}

func (c *CryptoSwap) FetchPoolFromRegistry(
	ctx context.Context,
	scanService *service.ScanService,
	cryptoPoolSourceAddress string,
	poolAddresses []common.Address,
) (
	twoPools []PoolItem,
	tricryptoPools []PoolItem,
	otherPools []common.Address,
	err error,
) {
	calls := make([]*repository.CallParams, 0, len(poolAddresses))
	coins := make([][8]common.Address, len(poolAddresses))
	decimals := make([][8]*big.Int, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		calls = append(
			calls, &repository.CallParams{
				ABI:    abis.CurveCryptoRegistry,
				Target: cryptoPoolSourceAddress,
				Method: "get_coins",
				Params: []interface{}{poolAddresses[i]},
				Output: &coins[i],
			},
		)
		calls = append(
			calls, &repository.CallParams{
				ABI:    abis.CurveCryptoRegistry,
				Target: cryptoPoolSourceAddress,
				Method: "get_decimals",
				Params: []interface{}{poolAddresses[i]},
				Output: &decimals[i],
			},
		)
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, nil, nil, err
	}

	tokens := make([][]PoolToken, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		var coinNumber int

		for j := 0; j < len(coins[i]); j++ {
			if coins[i][j].Hex() == constant.AddressZero {
				break
			}
			coinNumber = j + 1
		}

		tokens[i] = make([]PoolToken, coinNumber)
		for j := 0; j < coinNumber; j++ {
			tokens[i][j] = PoolToken{
				Address: strings.ToLower(coins[i][j].Hex()),
				Precision: new(big.Int).Div(
					constant.TenPowInt(18), constant.TenPowInt(uint8(decimals[i][j].Int64())),
				).String(),
				Rate: "",
			}
		}

		if coinNumber == 2 {
			twoPools = append(
				twoPools, PoolItem{
					ID:      strings.ToLower(poolAddresses[i].Hex()),
					Type:    constant.PoolTypes.CurveTwo,
					Version: 2,
					Tokens:  tokens[i],
				},
			)
		} else if coinNumber == 3 {
			tricryptoPools = append(
				tricryptoPools, PoolItem{
					ID:      strings.ToLower(poolAddresses[i].Hex()),
					Type:    constant.PoolTypes.CurveTricrypto,
					Version: 2,
					Tokens:  tokens[i],
				},
			)
		} else {
			otherPools = append(otherPools, poolAddresses[i])
		}
	}

	return twoPools, tricryptoPools, otherPools, nil
}

func (c *CryptoSwap) FetchPoolFromFactory(
	ctx context.Context,
	scanService *service.ScanService,
	cryptoPoolSourceAddress string,
	poolAddresses []common.Address,
) (
	twoPools []PoolItem,
	err error,
) {
	if cryptoPoolSourceAddress == constant.AddressZero {
		return twoPools, nil
	}
	calls := make([]*repository.CallParams, 0, len(poolAddresses))
	coins := make([][2]common.Address, len(poolAddresses))
	decimals := make([][2]*big.Int, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		calls = append(
			calls, &repository.CallParams{
				ABI:    abis.CurveCryptoFactory,
				Target: cryptoPoolSourceAddress,
				Method: "get_coins",
				Params: []interface{}{poolAddresses[i]},
				Output: &coins[i],
			},
		)
		calls = append(
			calls, &repository.CallParams{
				ABI:    abis.CurveCryptoFactory,
				Target: cryptoPoolSourceAddress,
				Method: "get_decimals",
				Params: []interface{}{poolAddresses[i]},
				Output: &decimals[i],
			},
		)
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	tokens := make([][]PoolToken, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		tokens[i] = make([]PoolToken, 2)
		for j := 0; j < len(coins[i]); j++ {
			tokens[i][j] = PoolToken{
				Address: strings.ToLower(coins[i][j].Hex()),
				Precision: new(big.Int).Div(
					constant.TenPowInt(18), constant.TenPowInt(uint8(decimals[i][j].Int64())),
				).String(),
				Rate: "",
			}
		}
		twoPools = append(
			twoPools, PoolItem{
				ID:      strings.ToLower(poolAddresses[i].Hex()),
				Type:    constant.PoolTypes.CurveTwo,
				Version: 2,
				Tokens:  tokens[i],
			},
		)
	}

	return twoPools, nil
}

func (c *CryptoSwap) AddCryptoPools(
	ctx context.Context,
	dex string,
	scanService *service.ScanService,
	poolType string,
	poolABI abi.ABI,
	poolItems []PoolItem,
) error {
	calls := make([]*repository.CallParams, 0, len(poolItems))
	lpTokens := make([]common.Address, len(poolItems))

	for i := 0; i < len(poolItems); i++ {
		calls = append(
			calls, &repository.CallParams{
				ABI:    poolABI,
				Target: poolItems[i].ID,
				Method: "token",
				Params: nil,
				Output: &lpTokens[i],
			},
		)
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i, poolItem := range poolItems {
		poolItem.LpToken = strings.ToLower(lpTokens[i].String())
		staticExtraBytes := c.curve.ExtractStaticExtra(ctx, scanService, poolItem)
		reserves, tokens, err := c.curve.ExtractReservesAndTokens(ctx, scanService, poolItem)
		if err != nil {
			logger.Errorf("can not extract reserves and tokens of a pool: %v", poolItem.ID)
			return fmt.Errorf("can not extract reserves and tokens of a pool: %v", poolItem.ID)
		}

		var newPool = entity.Pool{
			Address:     strings.ToLower(poolItem.ID),
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    dex,
			Type:        poolType,
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
