package fulcrom

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
)

type VaultScanner struct {
	config               *Config
	vaultReader          IVaultReader
	vaultPriceFeedReader IVaultPriceFeedReader
	usdgReader           IUSDGReader
	log                  logger.Logger
}

func NewVaultScanner(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *VaultScanner {
	return &VaultScanner{
		config:               config,
		vaultReader:          NewVaultReader(ethrpcClient),
		vaultPriceFeedReader: NewVaultPriceFeedReader(ethrpcClient),
		usdgReader:           NewUSDGReader(ethrpcClient),
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeFulcrom,
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

	if err != nil {
		return nil, err
	}

	return vaultPriceFeed, nil
}
