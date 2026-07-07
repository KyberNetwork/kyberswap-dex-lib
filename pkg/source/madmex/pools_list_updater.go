package madmex

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexTypeMadmex, NewPoolsListUpdater)

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
	byteValue, ok := bytesByPath[vaultPath]
	if !ok {
		return "", errors.New("misconfigured vault")
	}

	var vaultAddress VaultAddress
	if err := json.Unmarshal(byteValue, &vaultAddress); err != nil {
		return "", err
	}

	return vaultAddress.Vault, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": d.config.DexID,
		"kind":            "getNewPools",
	})
	if d.hasInitialized {
		log.Info("initialized. Ignore making new pools")
		return nil, nil, nil
	}

	vaultAddr, err := getVaultAddress(d.config.VaultPath)
	if err != nil {
		log.Errorf("get vault address failed: %v", err)
		return nil, nil, err
	}

	vault, err := NewVaultScanner(ChainID(d.config.ChainID), d.ethrpcClient).getVault(ctx, vaultAddr)
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
		Address:   vaultAddr,
		Exchange:  d.config.DexID,
		Type:      d.config.DexID,
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}

	d.hasInitialized = true
	log.Infof("got %v vault from file: %v", vaultAddr, d.config.VaultPath)

	return []entity.Pool{pool}, nil, nil
}
