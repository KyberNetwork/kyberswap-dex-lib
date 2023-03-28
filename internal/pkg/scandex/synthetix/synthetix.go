package synthetix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Synthetix struct {
	scanService *service.ScanService
	scanDexCfg  *config.ScanDex
	properties  Properties

	poolStateReader                       IPoolStateReader
	systemSettingsReader                  ISystemSettingsReader
	exchangerWithFeeRecAlternativesReader IExchangerWithFeeRecAlternativesReader
	exchangeRatesReader                   IExchangeRatesReader
	chainlinkDataFeedReader               IChainlinkDataFeedReader
	dexPriceAggregatorUniswapV3Reader     IDexPriceAggregatorUniswapV3Reader
}

func New(
	scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	chainId := scanService.Config().ChainID
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(chainId))

	if poolStateVersion == PoolStateVersionNormal {
		return &Synthetix{
			scanService: scanService,
			scanDexCfg:  scanDexCfg,
			properties:  properties,

			poolStateReader:         NewPoolStateReader(scanService),
			systemSettingsReader:    NewSystemSettingsReader(scanService),
			exchangeRatesReader:     NewExchangeRatesReader(scanService),
			chainlinkDataFeedReader: NewChainlinkDataFeedReader(scanService),
		}, nil
	}

	return &Synthetix{
		scanService: scanService,
		scanDexCfg:  scanDexCfg,
		properties:  properties,

		poolStateReader:                       NewPoolStateReader(scanService),
		systemSettingsReader:                  NewSystemSettingsReader(scanService),
		exchangerWithFeeRecAlternativesReader: NewExchangerWithFeeRecAlternativesReader(scanService),
		exchangeRatesReader:                   NewExchangeRatesWithDexPricingReader(scanService),
		chainlinkDataFeedReader:               NewChainlinkDataFeedReader(scanService),
		dexPriceAggregatorUniswapV3Reader:     NewDexPriceAggregatorUniswapV3Reader(scanService),
	}, nil
}

// InitPool ...
func (s *Synthetix) InitPool(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		logger.Infof("initialized pool in %v", time.Since(startTime))
	}()

	addresses, err := s.getAddresses()
	if err != nil {
		return err
	}

	poolState, err := s.getPoolState(ctx, addresses.Synthetix)
	if err != nil {
		return err
	}

	poolState.Addresses = addresses

	systemSettings, err := s.getSystemSettings(ctx, poolState)
	if err != nil {
		return err
	}

	poolState.SystemSettings = systemSettings

	poolState, err = s.getExchangerWithFeeRecAlternativesData(ctx, poolState)
	if err != nil {
		return err
	}

	poolState, err = s.getExchangeRates(ctx, poolState)
	if err != nil {
		return err
	}

	chainlinkNumRounds := s.getChainlinkNumRounds(poolState.SystemSettings.DynamicFeeConfig.Rounds)

	aggregators, err := s.getChainlinkDataFeeds(
		ctx,
		poolState.SUSDCurrencyKey,
		poolState.AggregatorAddresses,
		chainlinkNumRounds,
	)
	if err != nil {
		return err
	}

	poolState.Aggregators = aggregators

	dexPriceAggregatorUniswapV3, err := s.getDexPriceAggregatorUniswapV3(ctx, poolState)
	if err != nil {
		return err
	}

	poolState.DexPriceAggregator = dexPriceAggregatorUniswapV3

	pool, err := s.newPool(addresses.Synthetix, poolState)
	if err != nil {
		return err
	}

	s.scanService.SavePool(ctx, *pool)

	for _, token := range pool.Tokens {
		if _, err = s.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
			return err
		}
	}

	return nil
}

// UpdateNewPools do nothing
func (s *Synthetix) UpdateNewPools(ctx context.Context) {}

// UpdateReserves ...
func (s *Synthetix) UpdateReserves(ctx context.Context) {
	for {
		if err := s.updatePoolState(ctx); err != nil {
			logger.Errorf("updatePoolState failed, error: %v", err)
		}

		time.Sleep(s.properties.ReserveJobInterval.Duration)
	}
}

// UpdateTotalSupply do nothing
func (s *Synthetix) UpdateTotalSupply(ctx context.Context) {}

func (s *Synthetix) getAddresses() (*Addresses, error) {
	addressFilePath := path.Join(
		s.scanService.Config().DataFolder,
		s.properties.AddressesPath,
	)

	addressesFile, err := os.Open(addressFilePath)
	if err != nil {
		return nil, err
	}

	defer addressesFile.Close()

	addressesFileContent, err := io.ReadAll(addressesFile)
	if err != nil {
		return nil, err
	}

	var addresses Addresses
	if err = json.Unmarshal(addressesFileContent, &addresses); err != nil {
		return nil, err
	}

	return &addresses, nil
}

func (s *Synthetix) newPool(address string, poolState *PoolState) (*entity.Pool, error) {
	poolTokens := make([]*entity.PoolToken, 0, len(poolState.Synths))
	reserves := make([]string, 0, len(poolState.Synths))
	for _, currencyKey := range poolState.CurrencyKeys {
		synthAddress := poolState.Synths[currencyKey]
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   strings.ToLower(synthAddress.String()),
			Swappable: true,
		})
		reserves = append(reserves, poolState.SynthsTotalSupply[currencyKey].String())
	}

	extra := Extra{PoolState: poolState}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:  strings.ToLower(address),
		Exchange: s.scanDexCfg.Id,
		Type:     constant.PoolTypes.Synthetix,
		Tokens:   poolTokens,
		Reserves: reserves,
		Extra:    string(extraBytes),
	}, nil
}

func (s *Synthetix) updatePoolState(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		executionTime := time.Since(startTime)

		logger.
			WithFields(logger.Fields{
				"dex":               s.scanDexCfg.Id,
				"poolsUpdatedCount": 1,
				"duration":          executionTime.Milliseconds(),
			}).
			Info("finished UpdateReserves")

		metrics.HistogramScannerUpdateReservesDuration(executionTime, s.scanDexCfg.Id, 1)
	}()

	pools, err := s.scanService.GetPoolsByExchange(ctx, s.scanDexCfg.Id)
	if err != nil {
		return err
	}

	if len(pools) == 0 {
		return errors.New("no Synthetix pool found")
	}

	pool := pools[0]

	addresses, err := s.getAddresses()
	if err != nil {
		return err
	}

	poolState, err := s.getPoolState(ctx, pool.Address)
	if err != nil {
		return err
	}

	poolState.Addresses = addresses

	systemSettings, err := s.getSystemSettings(ctx, poolState)
	if err != nil {
		return err
	}

	poolState.SystemSettings = systemSettings

	poolState, err = s.getExchangerWithFeeRecAlternativesData(ctx, poolState)
	if err != nil {
		return err
	}

	poolState, err = s.getExchangeRates(ctx, poolState)
	if err != nil {
		return err
	}

	chainlinkNumRounds := s.getChainlinkNumRounds(poolState.SystemSettings.DynamicFeeConfig.Rounds)

	aggregators, err := s.getChainlinkDataFeeds(
		ctx,
		poolState.SUSDCurrencyKey,
		poolState.AggregatorAddresses,
		chainlinkNumRounds,
	)
	if err != nil {
		return err
	}

	poolState.Aggregators = aggregators

	dexPriceAggregatorUniswapV3, err := s.getDexPriceAggregatorUniswapV3(ctx, poolState)
	if err != nil {
		return err
	}

	poolState.DexPriceAggregator = dexPriceAggregatorUniswapV3

	poolTokens := make([]*entity.PoolToken, 0, len(poolState.Synths))
	reserves := make([]string, 0, len(poolState.Synths))
	for _, currencyKey := range poolState.CurrencyKeys {
		synthAddress := poolState.Synths[currencyKey]
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   strings.ToLower(synthAddress.String()),
			Swappable: true,
		})
		reserves = append(reserves, poolState.SynthsTotalSupply[currencyKey].String())
	}

	extra := Extra{PoolState: poolState}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal extra failed, pool: %s, err: %v", pool.Address, err)
	}

	pool.Extra = string(extraBytes)
	pool.Reserves = reserves
	pool.Tokens = poolTokens

	if err = s.scanService.SavePool(ctx, pool); err != nil {
		return fmt.Errorf("failed to save Synthetix pool, err: %v", err)
	}

	if err = s.scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), pool.Reserves); err != nil {
		return fmt.Errorf("failed to update Synthetix pool reserve, err: %v", err)
	}

	return nil
}

// ================================================================================

func (s *Synthetix) getPoolState(ctx context.Context, address string) (*PoolState, error) {
	poolState, err := s.poolStateReader.Read(ctx, address)
	if err != nil {
		return nil, err
	}

	return poolState, nil
}

func (s *Synthetix) getSystemSettings(ctx context.Context, poolState *PoolState) (*SystemSettings, error) {
	systemSettings, err := s.systemSettingsReader.Read(ctx, poolState)
	if err != nil {
		return nil, err
	}

	return systemSettings, nil
}

func (s *Synthetix) getExchangerWithFeeRecAlternativesData(
	ctx context.Context,
	poolState *PoolState,
) (*PoolState, error) {
	chainId := s.scanService.Config().ChainID
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(chainId))

	// Normal version does not have to fetch this data
	if poolStateVersion == PoolStateVersionNormal {
		return poolState, nil
	}

	poolState, err := s.exchangerWithFeeRecAlternativesReader.Read(ctx, poolState)
	if err != nil {
		return nil, err
	}

	return poolState, nil
}

func (s *Synthetix) getExchangeRates(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	poolState, err := s.exchangeRatesReader.Read(ctx, poolState)
	if err != nil {
		return nil, err
	}

	return poolState, nil
}

func (s *Synthetix) getChainlinkNumRounds(dynamicFeeConfigRounds *big.Int) *big.Int {
	var chainlinkNumRounds *big.Int
	// Choose bigger number of num rounds because we need to get all required data
	if dynamicFeeConfigRounds.Cmp(DefaultChainlinkNumRounds) > 0 {
		chainlinkNumRounds = dynamicFeeConfigRounds
	} else {
		chainlinkNumRounds = DefaultChainlinkNumRounds
	}

	return chainlinkNumRounds
}

func (s *Synthetix) getChainlinkDataFeeds(
	ctx context.Context,
	sUSDCurrencyKey string,
	aggregatorAddresses map[string]common.Address,
	numRound *big.Int,
) (map[string]*ChainlinkDataFeed, error) {
	roundCount := int(numRound.Int64())
	aggregators := make(map[string]*ChainlinkDataFeed, len(aggregatorAddresses))

	for aggregatorKey, aggregatorAddress := range aggregatorAddresses {
		if aggregatorKey == sUSDCurrencyKey {
			continue
		}

		aggregator, err := s.chainlinkDataFeedReader.Read(ctx, aggregatorAddress.String(), roundCount)
		if err != nil {
			return nil, err
		}

		aggregators[aggregatorKey] = aggregator
	}

	return aggregators, nil
}

func (s *Synthetix) getDexPriceAggregatorUniswapV3(
	ctx context.Context,
	poolState *PoolState,
) (*DexPriceAggregatorUniswapV3, error) {
	chainId := s.scanService.Config().ChainID
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(chainId))

	// Normal version does not have DexPriceAggregatorUniswapV3
	if poolStateVersion == PoolStateVersionNormal {
		return nil, nil
	}

	dexPriceAggregatorUniswapV3, err := s.dexPriceAggregatorUniswapV3Reader.Read(ctx, poolState)
	if err != nil {
		return nil, err
	}

	dexPriceAggregatorUniswapV3.BlockTimestamp = poolState.BlockTimestamp

	return dexPriceAggregatorUniswapV3, nil
}

func getPoolStateVersion(chainID valueobject.ChainID) PoolStateVersion {
	poolStateVersion, ok := PoolStateVersionByChainID[chainID]
	if !ok {
		return DefaultPoolStateVersion
	}

	return poolStateVersion
}
