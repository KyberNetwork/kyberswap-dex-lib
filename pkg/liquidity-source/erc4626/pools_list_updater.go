package erc4626

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger      logger.Logger
	initialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	lg := logger.WithFields(logger.Fields{
		"dexId":   cfg.DexId,
		"dexType": DexType,
	})

	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list.")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	if u.initialized {
		u.logger.Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	var errs []error
	pools := make([]entity.Pool, 0, len(u.cfg.Vaults))
	for vaultAddr, vaultCfg := range u.cfg.Vaults {
		pool, err := u.getNewPool(ctx, vaultAddr, vaultCfg)
		if err != nil {
			errs = append(errs, errors.WithMessage(err, vaultAddr))
		} else {
			pools = append(pools, *pool)
		}
	}

	if len(errs) > 0 {
		return nil, metadataBytes, errors.Errorf("failed to get new pools: %v", errs)
	}

	u.initialized = true

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getNewPool(ctx context.Context, vaultAddr string, vaultCfg VaultCfg) (*entity.Pool, error) {
	assetToken, state, err := fetchAssetAndState(ctx, u.ethrpcClient, vaultAddr, vaultCfg, true, nil)
	if err != nil {
		u.logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetchAssetAndState")
		return nil, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		IsNativeAsset: valueobject.IsNative(hexutil.Encode(assetToken[:])),
	})
	if err != nil {
		return nil, err
	}

	p := &entity.Pool{
		Address:  strings.ToLower(vaultAddr),
		Exchange: u.cfg.DexId,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(vaultAddr), Swappable: true},
			{Address: valueobject.WrapNativeLower(hexutil.Encode(assetToken[:]), u.cfg.ChainId), Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
	}

	return p, updateEntityState(p, vaultCfg, state)
}
