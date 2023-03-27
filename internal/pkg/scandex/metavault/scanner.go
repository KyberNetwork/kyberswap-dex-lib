package metavault

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

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/metrics"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type Scanner struct {
	scanService *service.ScanService
	scanDexCfg  *config.ScanDex
	properties  Properties

	vaultReader           IVaultReader
	vaultPriceFeedReader  IVaultPriceFeedReader
	fastPriceFeedV1Reader IFastPriceFeedV1Reader
	fastPriceFeedV2Reader IFastPriceFeedV2Reader
	priceFeedReader       IPriceFeedReader
	usdmReader            IUSDMReader
	chainlinkFlagsReader  IChainlinkFlagsReader
	pancakePairReader     IPancakePairReader
}

func New(
	scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	return &Scanner{
		scanService: scanService,
		scanDexCfg:  scanDexCfg,
		properties:  properties,

		vaultReader:           NewVaultReader(scanService),
		vaultPriceFeedReader:  NewVaultPriceFeedReader(scanService),
		fastPriceFeedV1Reader: NewFastPriceFeedV1Reader(scanService),
		fastPriceFeedV2Reader: NewFastPriceFeedV2Reader(scanService),
		priceFeedReader:       NewPriceFeedReader(scanService),
		usdmReader:            NewUSDMReader(scanService),
		chainlinkFlagsReader:  NewChainlinkFlagsReader(scanService),
		pancakePairReader:     NewPancakePairReader(scanService),
	}, nil
}

// InitPool ...
func (s *Scanner) InitPool(ctx context.Context) error {
	return nil
}

// UpdateNewPools do nothing
func (s *Scanner) UpdateNewPools(ctx context.Context) {
	startTime := time.Now()
	defer func() {
		logger.Infof("initialized pool in %v", time.Since(startTime))
	}()

	addresses, err := s.getAddresses()
	if err != nil {
		logger.Errorf("getAddresses failed: [%v]", err)
		return
	}

	vault, err := s.getVault(ctx, addresses.Vault)
	if err != nil {
		logger.Errorf("getVault failed: [%v]", err)
		return
	}

	pool, err := s.newPool(addresses.Vault, vault)
	if err != nil {
		logger.Errorf("newPool failed: [%v]", err)
		return
	}

	if err := s.scanService.SavePool(ctx, *pool); err != nil {
		logger.Errorf("SavePool failed: [%v]", err)
		return
	}

	for _, token := range pool.Tokens {
		if _, err = s.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
			logger.Errorf("FetchOrGetToken failed: [%v]", err)
			return
		}
	}
}

// UpdateReserves ...
func (s *Scanner) UpdateReserves(ctx context.Context) {
	for {
		if err := s.updateReserves(ctx); err != nil {
			logger.Errorf("updateReserves failed, error: %v", err)
		}

		time.Sleep(s.properties.ReserveJobInterval.Duration)
	}
}

// UpdateTotalSupply do nothing
func (s *Scanner) UpdateTotalSupply(ctx context.Context) {}

func (s *Scanner) getAddresses() (*Addresses, error) {
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

func (s *Scanner) newPool(address string, vault *Vault) (*entity.Pool, error) {
	poolTokens := make([]*entity.PoolToken, 0, len(vault.WhitelistedTokens))
	reserves := make([]string, 0, len(vault.WhitelistedTokens))
	for _, token := range vault.WhitelistedTokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Swappable: true,
		})
		reserves = append(reserves, vault.PoolAmounts[token].String())
	}

	extra := Extra{Vault: vault}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   strings.ToLower(address),
		Exchange:  s.scanDexCfg.Id,
		Type:      constant.PoolTypes.Metavault,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *Scanner) updateReserves(ctx context.Context) error {
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
		return errors.New("no metavault pool found")
	}

	pool := pools[0]

	vault, err := s.getVault(ctx, pool.Address)
	if err != nil {
		return fmt.Errorf("get vault failed, pool: %s, err: %v", pool.Address, err)
	}

	poolTokens := make([]*entity.PoolToken, 0, len(vault.WhitelistedTokens))
	reserves := make([]string, 0, len(vault.WhitelistedTokens))
	for _, token := range vault.WhitelistedTokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Swappable: true,
		})
		reserves = append(reserves, vault.PoolAmounts[token].String())
	}

	extra := Extra{Vault: vault}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return fmt.Errorf("marshal extra failed, pool: %s, err: %v", pool.Address, err)
	}

	pool.Extra = string(extraBytes)
	pool.Reserves = reserves
	pool.Tokens = poolTokens
	pool.Timestamp = time.Now().Unix()

	if err = s.scanService.SavePool(ctx, pool); err != nil {
		return fmt.Errorf("save pool failed, err: %v", err)
	}

	return nil
}

// ================================================================================

func (s *Scanner) getVault(ctx context.Context, address string) (*Vault, error) {
	vault, err := s.vaultReader.Read(ctx, address)
	if err != nil {
		return nil, err
	}

	usdm, err := s.usdmReader.Read(ctx, vault.USDMAddress.String())
	if err != nil {
		return nil, err
	}

	vault.USDM = usdm

	vaultPriceFeed, err := s.getVaultPriceFeed(ctx, vault.PriceFeedAddress.String(), vault.WhitelistedTokens)
	if err != nil {
		return nil, err
	}

	vault.PriceFeed = vaultPriceFeed

	return vault, nil
}

func (s *Scanner) getVaultPriceFeed(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error) {
	vaultPriceFeed, err := s.vaultPriceFeedReader.Read(ctx, address, tokens)
	if err != nil {
		return nil, err
	}

	secondaryPriceFeedVersion := getSecondaryPriceFeedVersion(s.scanService.Config().ChainID)

	vaultPriceFeed.SecondaryPriceFeedVersion = secondaryPriceFeedVersion

	fastPriceFeed, err := s.getFastPriceFeed(
		ctx,
		secondaryPriceFeedVersion,
		vaultPriceFeed.SecondaryPriceFeedAddress.String(),
		tokens,
	)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.SecondaryPriceFeed = fastPriceFeed

	priceFeeds, err := s.getPriceFeeds(ctx, vaultPriceFeed.PriceFeedsAddresses, vaultPriceFeed.PriceSampleSpace)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.PriceFeeds = priceFeeds

	return vaultPriceFeed, nil
}

func (s *Scanner) getPriceFeeds(
	ctx context.Context,
	priceFeedAddresses map[string]common.Address,
	priceSampleSpace *big.Int,
) (map[string]*PriceFeed, error) {
	roundCount := int(priceSampleSpace.Int64())
	priceFeeds := make(map[string]*PriceFeed, len(priceFeedAddresses))

	for tokenAddress, priceFeedAddress := range priceFeedAddresses {
		priceFeed, err := s.priceFeedReader.Read(ctx, priceFeedAddress.String(), roundCount)
		if err != nil {
			return nil, err
		}

		priceFeeds[tokenAddress] = priceFeed
	}

	return priceFeeds, nil
}

func (s *Scanner) getFastPriceFeed(
	ctx context.Context,
	version int,
	address string,
	tokens []string,
) (IFastPriceFeed, error) {
	if version == 2 {
		return s.fastPriceFeedV2Reader.Read(ctx, address, tokens)
	}

	return s.fastPriceFeedV1Reader.Read(ctx, address, tokens)
}

func getSecondaryPriceFeedVersion(chainID int) int {
	secondaryPriceFeedVersion, ok := SecondaryPriceFeedVersionByChainID[chainID]
	if !ok {
		return DefaultSecondaryPriceFeedVersion
	}

	return secondaryPriceFeedVersion
}
