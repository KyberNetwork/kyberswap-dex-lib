package gmx

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

type GMX struct {
	scanService *service.ScanService
	scanDexCfg  *config.ScanDex
	properties  Properties

	vaultReader           IVaultReader
	vaultPriceFeedReader  IVaultPriceFeedReader
	fastPriceFeedV1Reader IFastPriceFeedV1Reader
	fastPriceFeedV2Reader IFastPriceFeedV2Reader
	priceFeedReader       IPriceFeedReader
	usdgReader            IUSDGReader
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

	return &GMX{
		scanService: scanService,
		scanDexCfg:  scanDexCfg,
		properties:  properties,

		vaultReader:           NewVaultReader(scanService),
		vaultPriceFeedReader:  NewVaultPriceFeedReader(scanService),
		fastPriceFeedV1Reader: NewFastPriceFeedV1Reader(scanService),
		fastPriceFeedV2Reader: NewFastPriceFeedV2Reader(scanService),
		priceFeedReader:       NewPriceFeedReader(scanService),
		usdgReader:            NewUSDGReader(scanService),
		chainlinkFlagsReader:  NewChainlinkFlagsReader(scanService),
		pancakePairReader:     NewPancakePairReader(scanService),
	}, nil
}

// InitPool ...
func (g *GMX) InitPool(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		logger.Infof("initialized pool in %v", time.Since(startTime))
	}()

	addresses, err := g.getAddresses()
	if err != nil {
		return err
	}

	vault, err := g.getVault(ctx, addresses.Vault)
	if err != nil {
		return err
	}

	pool, err := g.newPool(addresses.Vault, vault)
	if err != nil {
		return err
	}

	g.scanService.SavePool(ctx, *pool)

	for _, token := range pool.Tokens {
		if _, err = g.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
			return err
		}
	}

	return nil
}

// UpdateNewPools do nothing
func (g *GMX) UpdateNewPools(ctx context.Context) {}

// UpdateReserves ...
func (g *GMX) UpdateReserves(ctx context.Context) {
	for {
		if err := g.updateReserves(ctx); err != nil {
			logger.Errorf("updateReserves failed, error: %v", err)
		}

		time.Sleep(g.properties.ReserveJobInterval.Duration)
	}
}

// UpdateTotalSupply do nothing
func (g *GMX) UpdateTotalSupply(ctx context.Context) {}

func (g *GMX) getAddresses() (*Addresses, error) {
	addressFilePath := path.Join(
		g.scanService.Config().DataFolder,
		g.properties.AddressesPath,
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

func (g *GMX) newPool(address string, vault *Vault) (*entity.Pool, error) {
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
		Exchange:  g.scanDexCfg.Id,
		Type:      constant.PoolTypes.GMX,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}, nil
}

func (g *GMX) updateReserves(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		executionTime := time.Since(startTime)

		logger.
			WithFields(logger.Fields{
				"dex":               g.scanDexCfg.Id,
				"poolsUpdatedCount": 1,
				"duration":          executionTime.Milliseconds(),
			}).
			Info("finished UpdateReserves")

		metrics.HistogramScannerUpdateReservesDuration(executionTime, g.scanDexCfg.Id, 1)
	}()

	pools, err := g.scanService.GetPoolsByExchange(ctx, g.scanDexCfg.Id)
	if err != nil {
		return err
	}

	if len(pools) == 0 {
		return errors.New("no gmx pool found")
	}

	pool := pools[0]

	vault, err := g.getVault(ctx, pool.Address)
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

	if err = g.scanService.SavePool(ctx, pool); err != nil {
		return fmt.Errorf("save pool failed, err: %v", err)
	}

	return nil
}

// ================================================================================

func (g *GMX) getVault(ctx context.Context, address string) (*Vault, error) {
	vault, err := g.vaultReader.Read(ctx, address)
	if err != nil {
		return nil, err
	}

	usdg, err := g.usdgReader.Read(ctx, vault.USDGAddress.String())
	if err != nil {
		return nil, err
	}

	vault.USDG = usdg

	vaultPriceFeed, err := g.getVaultPriceFeed(ctx, vault.PriceFeedAddress.String(), vault.WhitelistedTokens)
	if err != nil {
		return nil, err
	}

	vault.PriceFeed = vaultPriceFeed

	return vault, nil
}

func (g *GMX) getVaultPriceFeed(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error) {
	vaultPriceFeed, err := g.vaultPriceFeedReader.Read(ctx, address, tokens)
	if err != nil {
		return nil, err
	}

	if !isAddressZero(vaultPriceFeed.ChainlinkFlagsAddress) {
		chainlinkFlags, err := g.chainlinkFlagsReader.Read(ctx, vaultPriceFeed.ChainlinkFlagsAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.ChainlinkFlags = chainlinkFlags
	}

	if !isAddressZero(vaultPriceFeed.BNBBUSDAddress) {
		bnbBusd, err := g.pancakePairReader.Read(ctx, vaultPriceFeed.BNBBUSDAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.BNBBUSD = bnbBusd
	}

	if !isAddressZero(vaultPriceFeed.BTCBNBAddress) {
		btcBnb, err := g.pancakePairReader.Read(ctx, vaultPriceFeed.BTCBNBAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.BTCBNB = btcBnb
	}

	if !isAddressZero(vaultPriceFeed.ETHBNBAddress) {
		ethBnb, err := g.pancakePairReader.Read(ctx, vaultPriceFeed.ETHBNBAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.ETHBNB = ethBnb
	}

	secondaryPriceFeedVersion := getSecondaryPriceFeedVersion(g.scanService.Config().ChainID)

	vaultPriceFeed.SecondaryPriceFeedVersion = secondaryPriceFeedVersion

	fastPriceFeed, err := g.getFastPriceFeed(
		ctx,
		secondaryPriceFeedVersion,
		vaultPriceFeed.SecondaryPriceFeedAddress.String(),
		tokens,
	)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.SecondaryPriceFeed = fastPriceFeed

	priceFeeds, err := g.getPriceFeeds(ctx, vaultPriceFeed.PriceFeedsAddresses, vaultPriceFeed.PriceSampleSpace)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.PriceFeeds = priceFeeds

	return vaultPriceFeed, nil
}

func (g *GMX) getPriceFeeds(
	ctx context.Context,
	priceFeedAddresses map[string]common.Address,
	priceSampleSpace *big.Int,
) (map[string]*PriceFeed, error) {
	roundCount := int(priceSampleSpace.Int64())
	priceFeeds := make(map[string]*PriceFeed, len(priceFeedAddresses))

	for tokenAddress, priceFeedAddress := range priceFeedAddresses {
		priceFeed, err := g.priceFeedReader.Read(ctx, priceFeedAddress.String(), roundCount)
		if err != nil {
			return nil, err
		}

		priceFeeds[tokenAddress] = priceFeed
	}

	return priceFeeds, nil
}

func (g *GMX) getFastPriceFeed(
	ctx context.Context,
	version int,
	address string,
	tokens []string,
) (IFastPriceFeed, error) {
	if version == 2 {
		return g.fastPriceFeedV2Reader.Read(ctx, address, tokens)
	}

	return g.fastPriceFeedV1Reader.Read(ctx, address, tokens)
}

func getSecondaryPriceFeedVersion(chainID int) int {
	secondaryPriceFeedVersion, ok := SecondaryPriceFeedVersionByChainID[chainID]
	if !ok {
		return DefaultSecondaryPriceFeedVersion
	}

	return secondaryPriceFeedVersion
}

func isAddressZero(address common.Address) bool {
	return strings.EqualFold(address.String(), constant.AddressZero)
}
