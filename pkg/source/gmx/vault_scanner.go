package gmx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type VaultScanner struct {
	config                *Config
	vaultReader           IVaultReader
	vaultPriceFeedReader  IVaultPriceFeedReader
	fastPriceFeedV1Reader IFastPriceFeedV1Reader
	fastPriceFeedV2Reader IFastPriceFeedV2Reader
	priceFeedReader       IPriceFeedReader
	usdgReader            IUSDGReader
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
		usdgReader:            NewUSDGReader(ethrpcClient),
		chainlinkFlagsReader:  NewChainlinkFlagsReader(ethrpcClient),
		pancakePairReader:     NewPancakePairReader(ethrpcClient),
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmx,
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

	usdg, err := vs.usdgReader.Read(ctx, vault.USDGAddress.String())
	if err != nil {
		vs.log.Errorf("error when usdgReader read: %s", err)
		return nil, err
	}

	vault.USDG = usdg

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

	if !eth.IsZeroAddress(vaultPriceFeed.ChainlinkFlagsAddress) {
		chainlinkFlags, err := vs.chainlinkFlagsReader.Read(ctx, vaultPriceFeed.ChainlinkFlagsAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.ChainlinkFlags = chainlinkFlags
	}

	if !eth.IsZeroAddress(vaultPriceFeed.BNBBUSDAddress) {
		bnbBusd, err := vs.pancakePairReader.Read(ctx, vaultPriceFeed.BNBBUSDAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.BNBBUSD = bnbBusd
	}

	if !eth.IsZeroAddress(vaultPriceFeed.BTCBNBAddress) {
		btcBnb, err := vs.pancakePairReader.Read(ctx, vaultPriceFeed.BTCBNBAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.BTCBNB = btcBnb
	}

	if !eth.IsZeroAddress(vaultPriceFeed.ETHBNBAddress) {
		ethBnb, err := vs.pancakePairReader.Read(ctx, vaultPriceFeed.ETHBNBAddress.String())
		if err != nil {
			return nil, err
		}

		vaultPriceFeed.ETHBNB = ethBnb
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

func (vs VaultScanner) getFastPriceFeed(
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
