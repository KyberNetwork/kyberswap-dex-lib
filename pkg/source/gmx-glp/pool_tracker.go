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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypeGmxGlp,
		"poolAddress":     p.Address,
	})
	log.Info("Start getting new state of pool")

	vault, err := NewVaultScanner(d.config, d.ethrpcClient).getVault(ctx, d.config.VaultAddress)
	if err != nil {
		log.Errorf("get vault failed: %v", err)
		return entity.Pool{}, fmt.Errorf("get vault failed, pool: %s, err: %v", p.Address, err)
	}
	glpManager, err := NewGlpManagerScanner(d.config, d.ethrpcClient).getGlpManager(ctx, d.config.GlpManagerAddress)
	if err != nil {
		log.Errorf("get glpManager failed: %v", err)
		return entity.Pool{}, fmt.Errorf("get glpManager failed, pool: %s, err: %v", d.config.GlpManagerAddress, err)
	}
	yearnTokenVault, err := NewYearnTokenVaultScanner(d.config, d.ethrpcClient).getYearnTokenVaultScanner(ctx, d.config.YearnTokenVaultAddress)
	if err != nil {
		return entity.Pool{}, fmt.Errorf("get yearnTokenVault failed, pool: %s, err: %v", d.config.YearnTokenVaultAddress, err)
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
		log.Errorf("marshal extra failed: %v", err)
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Tokens = poolTokens
	p.Timestamp = time.Now().Unix()

	log.Info("Finish getting new state")

	return p, nil
}
