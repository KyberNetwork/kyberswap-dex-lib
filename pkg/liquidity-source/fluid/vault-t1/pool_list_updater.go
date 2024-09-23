package vaultT1

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	paths, err := u.getSwapPaths(ctx)

	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0)

	for _, swapPath := range paths {
		tokenInName, tokenInSymbol, tokenInDecimals, err := u.readTokenSymbolAndName(ctx, swapPath.TokenIn)
		if err != nil {
			return nil, nil, err
		}

		tokenOutName, tokenOutSymbol, tokenOutDecimals, err := u.readTokenSymbolAndName(ctx, swapPath.TokenOut)
		if err != nil {
			return nil, nil, err
		}

		pool := entity.Pool{
			Address:  swapPath.Protocol.String(),
			Exchange: string(valueobject.ExchangeFluidVaultT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   swapPath.TokenIn.String(),
					Name:      tokenInName,
					Symbol:    tokenInSymbol,
					Decimals:  tokenInDecimals,
					Weight:    1,
					Swappable: true,
				},
				{
					Address:   swapPath.TokenOut.String(),
					Name:      tokenOutName,
					Symbol:    tokenOutSymbol,
					Decimals:  tokenOutDecimals,
					Weight:    1,
					Swappable: false,
				},
			},
		}

		pools = append(pools, pool)
	}

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getSwapPaths(ctx context.Context) ([]SwapPath, error) {
	var paths []SwapPath

	req := u.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    vaultLiquidationResolverABI,
		Target: vaultLiquidationResolver[u.config.ChainID],
		Method: VLRMethodGetAllSwapPaths,
	}, []interface{}{&paths})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Error("aggregate request failed")
		return nil, err
	}

	return paths, nil
}

func (u *PoolsListUpdater) readTokenSymbolAndName(ctx context.Context, token common.Address) (string, string, uint8, error) {
	if token == common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE") {
		return "ETH", "Ethereum", 18, nil
	}

	var symbol, name string
	var decimals uint8

	req := u.ethrpcClient.R().SetContext(ctx)

	// Symbol call
	req.AddCall(&ethrpc.Call{
		ABI:    erc20,
		Target: token.String(),
		Method: TokenMethodSymbol,
		Params: nil,
	}, []interface{}{&symbol})

	// Name call
	req.AddCall(&ethrpc.Call{
		ABI:    erc20,
		Target: token.String(),
		Method: TokenMethodName,
		Params: nil,
	}, []interface{}{&name})

	// Decimals call
	req.AddCall(&ethrpc.Call{
		ABI:    erc20,
		Target: token.String(),
		Method: TokenMethodDecimals,
		Params: nil,
	}, []interface{}{&decimals})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("can not read token info")
		return "", "", 0, err
	}

	return name, symbol, decimals, nil
}
