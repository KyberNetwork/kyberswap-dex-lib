package metavault

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"

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

func getVaultAddress(vaultPath string) (string, error) {
	byteValue, ok := BytesByPath[vaultPath]
	if !ok {
		return "", errors.New("misconfigured vault")
	}

	var vaultAddress VaultAddress
	if err := sonic.Unmarshal(byteValue, &vaultAddress); err != nil {
		return "", err
	}

	return vaultAddress.Vault, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypeMetavault,
		"kind":            "getNewPools",
	})
	if d.hasInitialized {
		log.Infof("initialized. Ignore making new pools")
		return nil, nil, nil
	}

	vaultAddr, err := getVaultAddress(d.config.VaultPath)
	if err != nil {
		log.Errorf("get vault address failed: %v", err)
		return nil, nil, err
	}

	vault, err := NewVaultScanner(MATIC, d.ethrpcClient).getVault(ctx, vaultAddr)
	if err != nil {
		log.Errorf("get vault failed: %v", err)
		return nil, nil, err
	}

	poolTokens := make([]*entity.PoolToken, 0, len(vault.WhitelistedTokens))
	reserves := make([]string, 0, len(vault.WhitelistedTokens))
	for _, token := range vault.WhitelistedTokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Swappable: true,
		})
		reserves = append(reserves, vault.PoolAmounts[token].String())
	}

	extra := Extra{Vault: vault}

	extraBytes, err := sonic.Marshal(extra)
	if err != nil {
		log.Errorf("error when marshal extra: %v", err)
		return nil, nil, err
	}

	pool := entity.Pool{
		Address:   vaultAddr,
		Exchange:  d.config.DexID,
		Type:      DexTypeMetavault,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}

	d.hasInitialized = true
	log.Infof("got %v vault from file: %v", vaultAddr, d.config.VaultPath)

	return []entity.Pool{pool}, nil, nil
}
