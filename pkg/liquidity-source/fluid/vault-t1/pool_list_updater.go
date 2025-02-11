package vaultT1

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

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

	pools := make([]entity.Pool, 0, len(paths))

	for _, swapPath := range paths {
		staticExtraBytes, err := json.Marshal(&StaticExtra{
			VaultLiquidationResolver: u.config.VaultLiquidationResolver,
			HasNative: strings.EqualFold(swapPath.TokenIn.Hex(), valueobject.NativeAddress) ||
				strings.EqualFold(swapPath.TokenOut.Hex(), valueobject.NativeAddress),
		})
		if err != nil {
			return nil, nil, err
		}
		pool := entity.Pool{
			Address:  swapPath.Protocol.Hex(),
			Exchange: string(valueobject.ExchangeFluidVaultT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.WrapNativeLower(swapPath.TokenIn.Hex(), u.config.ChainID),
					Weight:    1,
					Swappable: true,
				},
				{
					Address:   valueobject.WrapNativeLower(swapPath.TokenOut.Hex(), u.config.ChainID),
					Weight:    1,
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
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
		Target: u.config.VaultLiquidationResolver,
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
