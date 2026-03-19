package usd_ai

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config         *Config
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, _ *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("usd-ai: skip since pool has been initialized")
		return nil, metadataBytes, nil
	}
	pool := d.buildPool()
	d.hasInitialized = true
	logger.WithFields(logger.Fields{"pool": pool.Address}).Info("usd-ai: pool built from config")
	return []entity.Pool{pool}, metadataBytes, nil
}

func (d *PoolsListUpdater) buildPool() entity.Pool {
	usdaiAddr := strings.ToLower(d.config.USDaiAddress)
	baseAddr := strings.ToLower(d.config.BaseTokenAddress)

	return entity.Pool{
		Address:   usdaiAddr,
		Exchange:  string(valueobject.ExchangeUsdAi),
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Tokens: []*entity.PoolToken{
			{Address: usdaiAddr, Swappable: true},
			{Address: baseAddr, Swappable: true},
		},
		Reserves:    entity.PoolReserves{defaultReserves, defaultReserves},
		StaticExtra: "{}",
		Extra:       "{}",
	}
}
