package swapbasedperp

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

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypeSwapBasedPerp,
		"kind":            "getNewPools",
	})
	if d.hasInitialized {
		log.Infof("initialized. Ignore making new pools")
		return nil, nil, nil
	}

	vault, err := NewVaultScanner(d.config, d.ethrpcClient).getVault(ctx, d.config.VaultAddress)
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

	extra := Extra{Vault: vault}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.Errorf("error when marshal extra: %v", err)
		return nil, nil, err
	}

	pool := entity.Pool{
		Address:   d.config.VaultAddress,
		Exchange:  d.config.DexID,
		Type:      DexTypeSwapBasedPerp,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}

	d.hasInitialized = true
	log.Infof("got %v vault", d.config.VaultAddress)

	return []entity.Pool{pool}, nil, nil
}
