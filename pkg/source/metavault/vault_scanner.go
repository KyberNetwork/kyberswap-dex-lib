package metavault

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type VaultScanner struct {
	chainID               ChainID
	vaultReader           IVaultReader
	vaultPriceFeedReader  IVaultPriceFeedReader
	fastPriceFeedV1Reader IFastPriceFeedV1Reader
	fastPriceFeedV2Reader IFastPriceFeedV2Reader
	priceFeedReader       IPriceFeedReader
	usdmReader            IUSDMReader
	log                   logger.Logger
}

func NewVaultScanner(
	chainID ChainID,
	ethrpcClient *ethrpc.Client,
) *VaultScanner {
	return &VaultScanner{
		chainID:               chainID,
		vaultReader:           NewVaultReader(ethrpcClient),
		vaultPriceFeedReader:  NewVaultPriceFeedReader(ethrpcClient),
		fastPriceFeedV1Reader: NewFastPriceFeedV1Reader(ethrpcClient),
		fastPriceFeedV2Reader: NewFastPriceFeedV2Reader(ethrpcClient),
		priceFeedReader:       NewPriceFeedReader(ethrpcClient),
		usdmReader:            NewUSDMReader(ethrpcClient),
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeMetavault,
			"scanner":         "VaultScanner",
		}),
	}
}

func (vs *VaultScanner) getVault(ctx context.Context, address string) (*Vault, error) {
	vault, err := vs.vaultReader.Read(ctx, address)
	if err != nil {
		vs.log.Errorf("error when vaultReader read: %s", err)
		return nil, err
	}

	usdm, err := vs.usdmReader.Read(ctx, vault.USDMAddress.String())
	if err != nil {
		vs.log.Errorf("error when usdmReader read: %s", err)
		return nil, err
	}

	vault.USDM = usdm

	vaultPriceFeed, err := vs.getVaultPriceFeed(ctx, vault.PriceFeedAddress.String(), vault.WhitelistedTokens)
	if err != nil {
		vs.log.Errorf("error when get vaultPriceFeed: %s", err)
		return nil, err
	}

	vault.PriceFeed = vaultPriceFeed

	return vault, nil
}

func (vs *VaultScanner) getVaultPriceFeed(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error) {
	vaultPriceFeed, err := vs.vaultPriceFeedReader.Read(ctx, address, tokens)
	if err != nil {
		return nil, err
	}

	secondaryPriceFeedVersion := getSecondaryPriceFeedVersion(vs.chainID)

	vaultPriceFeed.SecondaryPriceFeedVersion = int(secondaryPriceFeedVersion)

	fastPriceFeed, err := vs.getFastPriceFeed(
		ctx,
		secondaryPriceFeedVersion,
		vaultPriceFeed.SecondaryPriceFeedAddress.String(),
		tokens,
	)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.SecondaryPriceFeed = fastPriceFeed

	priceFeeds, err := vs.getPriceFeeds(ctx, vaultPriceFeed.PriceFeedsAddresses, vaultPriceFeed.PriceSampleSpace)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.PriceFeeds = priceFeeds

	return vaultPriceFeed, nil
}

func (vs *VaultScanner) getPriceFeeds(
	ctx context.Context,
	priceFeedAddresses map[string]common.Address,
	priceSampleSpace *big.Int,
) (map[string]*PriceFeed, error) {
	roundCount := int(priceSampleSpace.Int64())
	priceFeeds := make(map[string]*PriceFeed, len(priceFeedAddresses))

	for tokenAddress, priceFeedAddress := range priceFeedAddresses {
		priceFeed, err := vs.priceFeedReader.Read(ctx, priceFeedAddress.String(), roundCount)
		if err != nil {
			return nil, err
		}

		priceFeeds[tokenAddress] = priceFeed
	}

	return priceFeeds, nil
}

func (vs *VaultScanner) getFastPriceFeed(
	ctx context.Context,
	version SecondaryPriceFeedVersion,
	address string,
	tokens []string,
) (IFastPriceFeed, error) {
	if version == SecondaryPriceFeedVersion2 {
		return vs.fastPriceFeedV2Reader.Read(ctx, address, tokens)
	}

	return vs.fastPriceFeedV1Reader.Read(ctx, address, tokens)
}

func getSecondaryPriceFeedVersion(chainID ChainID) SecondaryPriceFeedVersion {
	secondaryPriceFeedVersion, ok := SecondaryPriceFeedVersionByChainID[chainID]
	if !ok {
		return DefaultSecondaryPriceFeedVersion
	}

	return secondaryPriceFeedVersion
}
