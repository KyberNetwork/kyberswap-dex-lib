package honey

import (
	"context"
	"errors"
	"strings"
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

func (d *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	pools, err := d.initPools()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools() ([]entity.Pool, error) {
	byteData, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, errors.New("misconfigured poolPath")
	}
	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, err
	}

	pools, err := d.processBatch(poolItems)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to processBatch")
		return nil, err
	}
	d.hasInitialized = true

	return pools, nil
}

func (d *PoolsListUpdater) processBatch(poolItems []PoolItem) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(poolItems))
	for _, pool := range poolItems {
		poolEntity, err := d.getNewPool(&pool)
		if err != nil {
			continue
		}
		pools = append(pools, poolEntity)
	}
	return pools, nil
}

func (d *PoolsListUpdater) getNewPool(pool *PoolItem) (entity.Pool, error) {
	tokens := make([]*entity.PoolToken, len(pool.Tokens))
	reserves := make(entity.PoolReserves, len(pool.Tokens))
	for i, token := range pool.Tokens {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: true,
		}
		reserves[i] = defaultReserves
	}

	poolEntity := entity.Pool{
		Address:   pool.ID,
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     "{}",
	}

	return poolEntity, nil
}
