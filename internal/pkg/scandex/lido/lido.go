package lido

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

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

const (
	WstETHLidoMethodStEthPerToken  = "stEthPerToken"
	WstETHLidoMethodTokensPerStEth = "tokensPerStEth"

	ERC20MethodTotalSupply = "totalSupply"
	ERC20MethodBalanceOf   = "balanceOf"

	ReserveZero = "0"
)

type Lido struct {
	properties  Properties
	scanDexCfg  *config.ScanDex
	scanService *service.ScanService
}

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}
	return &Lido{
		properties:  properties,
		scanDexCfg:  scanDexCfg,
		scanService: scanService,
	}, nil
}

func (t *Lido) InitPool(ctx context.Context) error {
	poolsFile, err := os.Open(path.Join(t.scanService.Config().DataFolder, t.properties.PoolPath))
	if err != nil {
		logger.Errorf("failed to open config file: %v", err)
		return err
	}

	byteValue, err := io.ReadAll(poolsFile)
	if err != nil {
		logger.Errorf("failed to read file: %v", err)
		return err
	}

	var pools []PoolItem
	err = json.Unmarshal(byteValue, &pools)
	if err != nil {
		logger.Errorf("failed to parse pools: %v", err)
		return err
	}

	for i := range pools {
		var pool = pools[i]
		if t.scanService.ExistPool(ctx, pool.ID) {
			continue
		}

		var tokens = make([]*entity.PoolToken, 0)
		var reserves = make(entity.PoolReserves, 0)

		for j, item := range pool.Tokens {
			tokenEntity := entity.PoolToken{
				Address:   strings.ToLower(item.Address),
				Name:      item.Name,
				Symbol:    item.Symbol,
				Decimals:  item.Decimals,
				Weight:    1,
				Swappable: true,
			}

			if _, err := t.scanService.FetchOrGetToken(ctx, pool.Tokens[j].Address); err != nil {
				return err
			}

			tokens = append(tokens, &tokenEntity)
			reserves = append(reserves, ReserveZero)
		}

		if len(pool.LpToken) == 0 {
			logger.Errorf("can not find lpToken of pool %v", pool.ID)
			return errors.New("can not find lpToken of pool " + pool.ID)
		}

		extra := Extra{
			TokensPerStEth: constant.Zero,
			StEthPerToken:  constant.Zero,
		}

		extraBytes, _ := json.Marshal(extra)

		var staticExtra = StaticExtra{
			LpToken: pool.LpToken,
		}

		staticExtraBytes, _ := json.Marshal(staticExtra)

		var newPool = entity.Pool{
			Address:     pool.ID,
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    t.scanDexCfg.Id,
			Type:        pool.Type,
			Timestamp:   0,
			Reserves:    reserves,
			Tokens:      tokens,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		err := t.scanService.SavePool(ctx, newPool)

		if err != nil {
			logger.Errorf("failed to InitPool, err: %+v", err)
		}
	}

	return nil
}

func (t *Lido) UpdateNewPools(ctx context.Context) {
	// DO NOTHING
}

func (t *Lido) UpdateReserves(ctx context.Context) {
	updateReserveFunc := func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
		updatedPools := 0

		for _, pool := range pools {
			extra, err := t.getPoolExtra(ctx, pool)
			if err != nil {
				continue
			}

			extraBytes, err := json.Marshal(extra)
			if err != nil {
				continue
			}

			// Save pool extra
			_ = t.scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))

			reserves, err := t.getPoolReserves(ctx, pool)
			if err != nil {
				continue
			}

			poolReserves := make(entity.PoolReserves, len(reserves))

			for i := range reserves {
				poolReserves[i] = reserves[i].String()
			}

			// Save pool reserve
			_ = t.scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), poolReserves)

			updatedPools++
		}

		return updatedPools
	}

	uniswap.UpdateReserveJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		updateReserveFunc,
		t.properties.ReserveJobInterval,
		t.properties.UpdateReserveBulk,
		t.properties.ConcurrentBatches,
	)
}

func (t *Lido) getPoolExtra(ctx context.Context, pool entity.Pool) (Extra, error) {
	var calls []*repository.CallParams
	var stEthPerToken, tokensPerStEth *big.Int

	callParamsFactory := repository.CallParamsFactory(abis.LidoWstETH, pool.Address)

	calls = append(
		calls,
		callParamsFactory(WstETHLidoMethodStEthPerToken, &stEthPerToken, nil),
		callParamsFactory(WstETHLidoMethodTokensPerStEth, &tokensPerStEth, nil),
	)

	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return Extra{}, err
	}

	extra := Extra{
		StEthPerToken:  stEthPerToken,
		TokensPerStEth: tokensPerStEth,
	}

	return extra, nil
}

func (t *Lido) getPoolReserves(ctx context.Context, pool entity.Pool) ([]*big.Int, error) {
	var reserves = make([]*big.Int, len(pool.Tokens))
	var calls = make([]*repository.CallParams, 0)
	for i := range pool.Tokens {
		// if token is wstETH, we get the balance by using `totalSupply` of the wstETH contract
		if pool.Tokens[i].Address == pool.GetLpToken() {
			calls = append(
				calls, &repository.CallParams{
					ABI:    abis.ERC20,
					Target: pool.Tokens[i].Address,
					Method: ERC20MethodTotalSupply,
					Params: nil,
					Output: &reserves[i],
				},
			)
		} else {
			// if token is stETH, we get the balance by using `balanceOf` from the stETH contract
			calls = append(
				calls, &repository.CallParams{
					ABI:    abis.ERC20,
					Target: pool.Tokens[i].Address,
					Method: ERC20MethodBalanceOf,
					Params: []interface{}{common.HexToAddress(pool.Address)},
					Output: &reserves[i],
				},
			)
		}
	}

	if err := t.scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	return reserves, nil
}

func (t *Lido) UpdateTotalSupply(ctx context.Context) {
	uniswap.UpdateTotalSupplyJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		uniswap.UpdateTotalSupplyHandler,
		t.properties.TotalSupplyJobIntervalSec,
		t.properties.UpdateReserveBulk,
	)
}
