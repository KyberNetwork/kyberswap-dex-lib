package fxdx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (p *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquidityType": DexTypeFxdx,
	})
	if p.hasInitialized {
		log.Infof("initialized. Ignore making new pools")
		return nil, nil, nil
	}

	vault, err := NewVaultScanner(p.config, p.ethrpcClient).getVault(ctx, p.config.VaultAddress)
	if err != nil {
		log.Errorf("get vault failed: %v", err)
		return nil, nil, err
	}

	poolTokens := make([]*entity.PoolToken, 0, len(vault.WhitelistedTokens))
	reserves := make(entity.PoolReserves, 0, len(vault.WhitelistedTokens))
	for _, token := range vault.WhitelistedTokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Swappable: true,
		})
		reserves = append(reserves, vault.PoolAmounts[token].String())
	}

	feeUtils, err := NewFeeUtilsV2Reader(p.ethrpcClient).Read(ctx, vault)
	if err != nil {
		log.Errorf("get fee utils v2 failed: %v", err)
		return nil, nil, err
	}

	extra := Extra{Vault: vault, FeeUtils: feeUtils}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.Errorf("error when marshal extra: %v", err)
		return nil, nil, err
	}

	pool := entity.Pool{
		Address:   p.config.VaultAddress,
		Exchange:  p.config.DexID,
		Type:      DexTypeFxdx,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}

	p.hasInitialized = true
	log.Infof("got %v vault", p.config.VaultAddress)

	return []entity.Pool{pool}, nil, nil
}
