package dai_usds

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

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
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
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
	var pools = make([]entity.Pool, 0, len(poolItems))

	for _, pool := range poolItems {
		var err error
		var poolEntity entity.Pool

		poolEntity, err = d.getNewPool(&pool)

		if err != nil {
			return nil, err
		}

		pools = append(pools, poolEntity)
	}

	return pools, nil
}

func (d *PoolsListUpdater) getNewPool(pool *PoolItem) (entity.Pool, error) {
	var tokens = make([]*entity.PoolToken, 0, len(pool.Tokens))
	var reserves = make(entity.PoolReserves, 0, len(pool.Tokens))
	req := d.ethrpcClient.R()
	for _, token := range pool.Tokens {
		tokenEntity := entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Name:      token.Name,
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}

		tokens = append(tokens, &tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	poolEntity := entity.Pool{
		Address:     pool.ID,
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		Extra:       "{}",
		StaticExtra: "{}",
	}

	return poolEntity, nil
}
