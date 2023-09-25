package metavault

import (
	"context"
	"encoding/json"
	"fmt"
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
		"liquiditySource": DexTypeMetavault,
		"poolAddress":     p.Address,
	})
	log.Infof("Start getting new state of pool")

	vault, err := NewVaultScanner(MATIC, d.ethrpcClient).getVault(ctx, p.Address)
	if err != nil {
		log.Errorf("get vault failed: %v", err)
		return entity.Pool{}, fmt.Errorf("get vault failed, pool: %s, err: %v", p.Address, err)
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
		log.Errorf("marshal extra failed: %v", err)
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Tokens = poolTokens
	p.Timestamp = time.Now().Unix()

	log.Infof("Finish getting new state")

	return p, nil
}
