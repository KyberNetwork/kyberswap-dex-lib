package gmxglp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		"liquiditySource": DexTypeGmxGlp,
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
	glpManager, err := NewGlpManagerScanner(d.config, d.ethrpcClient).getGlpManager(ctx, d.config.GlpManagerAddress)
	if err != nil {
		log.Errorf("get glpManager failed: %v", err)
		return nil, nil, fmt.Errorf("get glpManager failed, pool: %s, err: %v", d.config.GlpManagerAddress, err)
	}
	yearnTokenVault, err := NewYearnTokenVaultScanner(d.config, d.ethrpcClient).getYearnTokenVaultScanner(ctx, d.config.YearnTokenVaultAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("get yearnTokenVault failed, pool: %s, err: %v", d.config.YearnTokenVaultAddress, err)
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

	// Add glpToken
	poolTokens = append(poolTokens, &entity.PoolToken{
		Address:   strings.ToLower(yearnTokenVault.Address),
		Swappable: true,
	})
	reserves = append(reserves, yearnTokenVault.TotalSupply.String())

	extra := Extra{
		Vault:           vault,
		GlpManager:      glpManager,
		YearnTokenVault: yearnTokenVault,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		log.Errorf("error when marshal extra: %v", err)
		return nil, nil, err
	}

	pool := entity.Pool{
		Address:   d.config.RewardRouterAddress,
		Exchange:  d.config.DexID,
		Type:      DexTypeGmxGlp,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}

	d.hasInitialized = true
	log.Infof("got %v vault", d.config.RewardRouterAddress)

	return []entity.Pool{pool}, nil, nil
}
