package platypus

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/metrics"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrInvalidOracleType = errors.New("invalid oracle type")
)

// IPoolSubgraphReader reads pool data from subgraph
type IPoolSubgraphReader interface {
	// GetPoolAddresses Gets all Platypus pool addresses
	GetPoolAddresses(ctx context.Context) ([]string, error)
}

// IPoolSCReader reads pool data from smart contract
type IPoolSCReader interface {
	// IsPaused gets pause status of the pool
	IsPaused(ctx context.Context, address string) (bool, error)
	// Read gets pool data
	Read(ctx context.Context, address string, fields ...PoolSCField) (PoolState, error)
	// GetAssetAddresses gets asset addresses of pool tokens
	GetAssetAddresses(ctx context.Context, address string, tokenAddresses []common.Address) ([]common.Address, error)
}

// IAssetSCReader reads asset data from smart contract
type IAssetSCReader interface {
	// BulkRead gets assets data
	BulkRead(ctx context.Context, addresses []string, fields ...AssetSCField) ([]AssetState, error)
}

// IStakedAvaxSCReader reads data from avax oracle smart contract
type IStakedAvaxSCReader interface {
	GetSAvaxRate(ctx context.Context, address string) (*big.Int, error)
}

type Platypus struct {
	properties         Properties
	scanDexCfg         *config.ScanDex
	scanService        *service.ScanService
	poolSubgraphReader IPoolSubgraphReader
	poolSCReader       IPoolSCReader
	assetSCReader      IAssetSCReader
	stakedAvaxSCReader IStakedAvaxSCReader
}

func New(
	scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	poolSubgraphReader := NewPoolSubgraphReader(
		properties.SubgraphAPI,
		PoolSubgraphReaderConfig{
			GetPoolAddressesBulk: properties.GetPoolAddressesBulk,
		},
	)

	return &Platypus{
		properties:         properties,
		scanDexCfg:         scanDexCfg,
		scanService:        scanService,
		poolSubgraphReader: poolSubgraphReader,
		poolSCReader:       NewPoolSCReader(scanService),
		assetSCReader:      NewAssetSCReader(scanService),
		stakedAvaxSCReader: NewStakedAvaxSCReader(scanService),
	}, nil
}

func (t *Platypus) InitPool(ctx context.Context) error {
	var count int
	startTime := time.Now()
	logger.Infof("init pool")
	defer func() {
		logger.Infof("initialized %d pool(s) in %v", count, time.Since(startTime))
	}()

	poolAddresses, err := t.poolSubgraphReader.GetPoolAddresses(ctx)
	if err != nil {
		return err
	}

	for _, poolAddress := range poolAddresses {
		isPaused, err := t.poolSCReader.IsPaused(ctx, poolAddress)
		if err != nil {
			return err
		}

		if isPaused {
			continue
		}

		pool, err := t.initPool(ctx, poolAddress)
		if err != nil {
			return err
		}

		for _, token := range pool.Tokens {
			if _, err := t.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
				return err
			}
		}

		if err := t.scanService.SavePool(ctx, *pool); err != nil {
			logger.Errorf("save pool failed, err: %v", err)
		}

		count++
	}

	return nil
}

func (t *Platypus) UpdateNewPools(ctx context.Context) {
	for {
		if err := t.updateNewPools(ctx); err != nil {
			logger.Errorf("updateNewPools failed, error: %v", err)
		}

		if err := t.updateExistingPools(ctx); err != nil {
			logger.Errorf("updateExistingPools failed, error: %v", err)
		}

		time.Sleep(time.Duration(t.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (t *Platypus) UpdateReserves(ctx context.Context) {
	for {
		if err := t.updateReserves(ctx); err != nil {
			logger.Infof("updateReserves failed, error: %v", err)
		}

		time.Sleep(t.properties.ReserveJobInterval.Duration)
	}
}

func (t *Platypus) UpdateTotalSupply(ctx context.Context) {
}

// updateNewPools calls subgraph to fetch all pool addresses
// it skips all exist or paused pools, the rest of pool will be initialized and saved
func (t *Platypus) updateNewPools(ctx context.Context) error {
	var count int
	startTime := time.Now()
	logger.Infof("update new pools")
	defer func() {
		logger.Infof("updated %d pool(s) in %v", count, time.Since(startTime))
	}()

	poolAddresses, err := t.poolSubgraphReader.GetPoolAddresses(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, poolAddress := range poolAddresses {
		wg.Add(1)

		go func(thePoolAddress string) {
			defer wg.Done()

			if t.scanService.ExistPool(ctx, thePoolAddress) {
				return
			}

			isPaused, err := t.poolSCReader.IsPaused(ctx, thePoolAddress)
			if err != nil {
				logger.Errorf("read paused failed, pool: %s, err: %v", thePoolAddress, err)
				return
			}

			if isPaused {
				return
			}

			pool, err := t.initPool(ctx, thePoolAddress)
			if err != nil {
				logger.Errorf("init pool failed, pool: %s, err: %v", thePoolAddress, err)
				return
			}

			for _, token := range pool.Tokens {
				if _, err := t.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
					logger.Errorf("fetch or get token failed, pool: %s, token: %s, err: %v", thePoolAddress, token.Address, err)
					return
				}
			}

			if err := t.scanService.SavePool(ctx, *pool); err != nil {
				logger.Errorf("save pool failed, err: %v", err)
			}
		}(poolAddress)

		count++
	}
	wg.Wait()

	return nil
}

// updateExistingPools updates pool tokens, assets and extra data
func (t *Platypus) updateExistingPools(ctx context.Context) error {
	var count int
	startTime := time.Now()
	logger.Infof("update existing pools")
	defer func() {
		logger.Infof("updated %d pool(s) in %v", count, time.Since(startTime))
	}()

	pools, err := t.scanService.GetPoolsByExchange(ctx, t.scanDexCfg.Id)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, pool := range pools {
		wg.Add(1)

		go func(thePool entity.Pool) {
			defer wg.Done()

			updatedPool, err := t.updateExistingPool(ctx, thePool)
			if err != nil {
				logger.Errorf("update existing pool failed, pool: %s, err: %v", updatedPool.Address, err)
				return
			}

			for _, token := range updatedPool.Tokens {
				if _, err := t.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
					logger.Errorf("fetch or get token failed, pool: %s, token: %s, err: %v", updatedPool.Address, token.Address, err)
					return
				}
			}

			if err := t.scanService.SavePool(ctx, updatedPool); err != nil {
				logger.Errorf("save pool failed, err: %v", err)
			}
			count++
		}(pool)
	}
	wg.Wait()

	return nil
}

// updateReserves updates data of existing pools
func (t *Platypus) updateReserves(ctx context.Context) error {
	var count int
	startTime := time.Now()
	defer func() {
		executionTime := time.Since(startTime)

		logger.
			WithFields(logger.Fields{
				"dex":               t.scanDexCfg.Id,
				"poolsUpdatedCount": count,
				"duration":          executionTime.Milliseconds(),
			}).
			Info("finished UpdateReserves")

		metrics.HistogramScannerUpdateReservesDuration(executionTime, t.scanDexCfg.Id, count)
	}()

	pools, err := t.scanService.GetPoolsByExchange(ctx, t.scanDexCfg.Id)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, pool := range pools {
		wg.Add(1)

		go func(thePool entity.Pool) {
			defer wg.Done()

			updatedPool, err := t.updatePoolReserves(ctx, thePool)
			if err != nil {
				logger.Errorf("update reserve failed, pool: %s, err: %v", thePool.Address, err)
				return
			}

			if err := t.scanService.SavePool(ctx, updatedPool); err != nil {
				logger.Errorf("save pool failed, err: %v", err)
			}
		}(pool)

		count++
	}
	wg.Wait()

	return nil
}

// initPool receives a poolAddress, fetch pool data from smart contracts
// then construct and return entity.Pool object
func (t *Platypus) initPool(ctx context.Context, poolAddress string) (*entity.Pool, error) {
	poolState, err := t.poolSCReader.Read(ctx, poolAddress, PoolSCFieldsToRead...)
	if err != nil {
		return nil, err
	}

	staticExtra := NewStaticExtra(poolState)

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, err
	}

	assetAddresses, err := t.poolSCReader.GetAssetAddresses(ctx, poolAddress, poolState.TokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenAddressesStr := make([]string, 0, len(poolState.TokenAddresses))
	for _, tokenAddress := range poolState.TokenAddresses {
		tokenAddressesStr = append(tokenAddressesStr, strings.ToLower(tokenAddress.Hex()))
	}

	assetAddressesStr := make([]string, 0, len(assetAddresses))
	for _, assetAddress := range assetAddresses {
		assetAddressesStr = append(assetAddressesStr, strings.ToLower(assetAddress.Hex()))
	}

	assetStates, err := t.assetSCReader.BulkRead(ctx, assetAddressesStr, AssetSCFieldsToRead...)
	if err != nil {
		return nil, err
	}

	poolType, err := NewPoolType(staticExtra.OracleType)
	if err != nil {
		return nil, err
	}

	var sAvaxRate *big.Int
	if poolType == constant.PoolTypes.PlatypusAvax {
		sAvaxRate, err = t.stakedAvaxSCReader.GetSAvaxRate(ctx, AddressStakedAvax)
		if err != nil {
			return nil, err
		}
	}

	extra := NewExtra(poolState, assetStates, sAvaxRate)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	reserves := make([]string, 0, len(tokenAddressesStr))
	for _, tokenAddress := range tokenAddressesStr {
		reserves = append(reserves, extra.AssetByToken[tokenAddress].Cash.String())
	}

	return &entity.Pool{
		Address:     poolAddress,
		ReserveUsd:  0,
		SwapFee:     0,
		Exchange:    t.scanDexCfg.Id,
		Type:        poolType,
		Timestamp:   time.Now().Unix(),
		Tokens:      NewPoolTokens(tokenAddressesStr),
		Reserves:    reserves,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		TotalSupply: "",
	}, nil
}

// updateExistingPool update pool tokens, assets and extra data
func (t *Platypus) updateExistingPool(ctx context.Context, pool entity.Pool) (entity.Pool, error) {
	poolState, err := t.poolSCReader.Read(ctx, pool.Address, PoolSCFieldsToRead...)
	if err != nil {
		return pool, err
	}

	var extra Extra
	if err = json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
		return pool, err
	}

	if poolState.Paused {
		extra.Paused = poolState.Paused

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return pool, err
		}

		pool.Extra = string(extraBytes)

		return pool, nil
	}

	assetAddresses, err := t.poolSCReader.GetAssetAddresses(ctx, pool.Address, poolState.TokenAddresses)
	if err != nil {
		return pool, err
	}

	tokenAddressesStr := make([]string, 0, len(poolState.TokenAddresses))
	for _, tokenAddress := range poolState.TokenAddresses {
		tokenAddressesStr = append(tokenAddressesStr, strings.ToLower(tokenAddress.Hex()))
	}

	assetAddressesStr := make([]string, 0, len(assetAddresses))
	for _, assetAddress := range assetAddresses {
		assetAddressesStr = append(assetAddressesStr, assetAddress.Hex())
	}

	assetStates, err := t.assetSCReader.BulkRead(ctx, assetAddressesStr, AssetSCFieldsToRead...)
	if err != nil {
		return pool, err
	}

	var sAvaxRate *big.Int
	if pool.Type == constant.PoolTypes.PlatypusAvax {
		sAvaxRate, err = t.stakedAvaxSCReader.GetSAvaxRate(ctx, AddressStakedAvax)
		if err != nil {
			return pool, err
		}
	}

	newExtra := NewExtra(poolState, assetStates, sAvaxRate)

	extraBytes, err := json.Marshal(newExtra)
	if err != nil {
		return pool, err
	}

	reserves := make([]string, 0, len(tokenAddressesStr))
	for _, tokenAddress := range tokenAddressesStr {
		reserves = append(reserves, extra.AssetByToken[tokenAddress].Cash.String())
	}

	pool.Tokens = NewPoolTokens(tokenAddressesStr)
	pool.Extra = string(extraBytes)
	pool.Reserves = reserves
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

// updatePoolReserves update pool assets cash and liability
func (t *Platypus) updatePoolReserves(ctx context.Context, pool entity.Pool) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
		return pool, err
	}

	assetAddresses := make([]string, 0, len(extra.AssetByToken))
	for _, asset := range extra.AssetByToken {
		assetAddresses = append(assetAddresses, asset.Address)
	}

	assetStates, err := t.assetSCReader.BulkRead(ctx, assetAddresses, AssetSCFieldsToRead...)
	if err != nil {
		return pool, err
	}

	assetByToken := make(map[string]Asset, len(assetStates))
	for _, assetState := range assetStates {
		assetByToken[strings.ToLower(assetState.UnderlyingToken.Hex())] = Asset{
			Address:          strings.ToLower(assetState.Address),
			Decimals:         assetState.Decimals,
			Cash:             assetState.Cash,
			Liability:        assetState.Liability,
			UnderlyingToken:  strings.ToLower(assetState.UnderlyingToken.Hex()),
			AggregateAccount: strings.ToLower(assetState.AggregateAccount.Hex()),
		}
	}

	extra.AssetByToken = assetByToken

	if pool.Type == constant.PoolTypes.PlatypusAvax {
		sAvaxRate, err := t.stakedAvaxSCReader.GetSAvaxRate(ctx, AddressStakedAvax)
		if err != nil {
			return pool, err
		}

		extra.SAvaxRate = sAvaxRate
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	reserves := make([]string, 0, len(pool.Tokens))
	for _, token := range pool.Tokens {
		reserves = append(reserves, extra.AssetByToken[token.Address].Cash.String())
	}

	pool.Extra = string(extraBytes)
	pool.Reserves = reserves

	return pool, nil
}

func NewPoolTokens(tokenAddresses []string) entity.PoolTokens {
	poolTokens := make([]*entity.PoolToken, 0, len(tokenAddresses))

	for _, tokenAddress := range tokenAddresses {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   tokenAddress,
			Swappable: true,
		})
	}

	return poolTokens
}

func NewPoolType(oracleType OracleType) (string, error) {
	switch oracleType {
	case OracleTypeChainlink:
		return constant.PoolTypes.PlatypusBase, nil
	case OracleTypeStakedAvax:
		return constant.PoolTypes.PlatypusAvax, nil
	case OracleTypeNone:
		return constant.PoolTypes.PlatypusPure, nil
	default:
		return "", ErrInvalidOracleType
	}
}
