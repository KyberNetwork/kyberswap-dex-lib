package genericarm

import (
	"context"
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

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}
	pools := make([]entity.Pool, 0, len(d.config.Arms))
	for armAddr, armCfg := range d.config.Arms {
		pool, err := d.getNewPool(ctx, armAddr, armCfg)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, *pool)
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")
	d.hasInitialized = true
	return pools, nil, nil
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, armAddr string, armCfg ArmCfg) (*entity.Pool, error) {
	poolState, err := fetchAssetAndState(ctx, d.ethrpcClient, armAddr, armCfg)
	if err != nil {
		return nil, err
	}

	extraBytes, err := json.Marshal(buildExtra(poolState, armCfg))
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal extra")
		return nil, err
	}

	tokens, reserves := buildTokensAndReserves(poolState, armCfg)
	return &entity.Pool{
		Address:   armAddr,
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     string(extraBytes),
	}, nil
}
