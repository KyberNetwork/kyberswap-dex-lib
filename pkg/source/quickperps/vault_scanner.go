package quickperps

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type VaultScanner struct {
	config                *Config
	vaultReader           IVaultReader
	vaultPriceFeedReader  IVaultPriceFeedReader
	fastPriceFeedV1Reader IFastPriceFeedV1Reader
	fastPriceFeedV2Reader IFastPriceFeedV2Reader
	priceFeedReader       IPriceFeedReader
	usdqReader            IUSDQReader
	chainlinkFlagsReader  IChainlinkFlagsReader
	pancakePairReader     IPancakePairReader
	log                   logger.Logger
}

func NewVaultScanner(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *VaultScanner {
	return &VaultScanner{
		config:                config,
		vaultReader:           NewVaultReader(ethrpcClient),
		vaultPriceFeedReader:  NewVaultPriceFeedReader(ethrpcClient),
		fastPriceFeedV1Reader: NewFastPriceFeedV1Reader(ethrpcClient),
		fastPriceFeedV2Reader: NewFastPriceFeedV2Reader(ethrpcClient),
		priceFeedReader:       NewPriceFeedReader(ethrpcClient),
		usdqReader:            NewUSDQReader(ethrpcClient),
		chainlinkFlagsReader:  NewChainlinkFlagsReader(ethrpcClient),
		pancakePairReader:     NewPancakePairReader(ethrpcClient),
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeQuickperps,
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

	usdq, err := vs.usdqReader.Read(ctx, vault.USDQAddress.String())
	if err != nil {
		vs.log.Errorf("error when usdqReader read: %s", err)
		return nil, err
	}

	vault.USDQ = usdq

	vaultPriceFeed, err := vs.getVaultPriceFeed(ctx, vault.PriceFeedAddress.String(), vault.WhitelistedTokens)
	if err != nil {
		vs.log.Errorf("error when get vaultPriceFeed: %s", err)
		return nil, err
	}

	vault.PriceFeed = vaultPriceFeed

	return vault, nil
}

// ================================================================================

func (vs *VaultScanner) getVaultPriceFeed(ctx context.Context, address string, tokens []string) (*VaultPriceFeed, error) {
	vaultPriceFeed, err := vs.vaultPriceFeedReader.Read(ctx, address, tokens)
	if err != nil {
		return nil, err
	}

	secondaryPriceFeedVersion := vs.getSecondaryPriceFeedVersion()

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

	vaultPriceFeed.SecondaryPriceFeed = NewIFastPriceFeedWrapper(fastPriceFeed)

	priceFeeds, err := vs.getPriceFeeds(ctx, vaultPriceFeed.PriceFeedsAddresses)
	if err != nil {
		return nil, err
	}

	vaultPriceFeed.PriceFeedProxies = priceFeeds

	return vaultPriceFeed, nil
}

func (vs *VaultScanner) getPriceFeeds(
	ctx context.Context,
	priceFeedAddresses map[string]common.Address,
) (map[string]*PriceFeed, error) {
	priceFeeds := make(map[string]*PriceFeed, len(priceFeedAddresses))

	for tokenAddress, priceFeedAddress := range priceFeedAddresses {
		priceFeed, err := vs.priceFeedReader.Read(ctx, priceFeedAddress.String())
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
	if version == secondaryPriceFeedVersion2 {
		return vs.fastPriceFeedV2Reader.Read(ctx, address, tokens)
	}

	return vs.fastPriceFeedV1Reader.Read(ctx, address, tokens)
}

func (vs *VaultScanner) getSecondaryPriceFeedVersion() SecondaryPriceFeedVersion {
	if vs.config.UseSecondaryPriceFeedV1 {
		return secondaryPriceFeedVersion1
	}
	return secondaryPriceFeedVersion2
}
