package hashflow

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
)

type PoolsListUpdater struct {
	config *Config
	client IClient
}

func NewPoolsListUpdater(cfg *Config, client IClient) *PoolsListUpdater {
	return &PoolsListUpdater{
		config: cfg,
		client: client,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	pools, err := d.initPools(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, metadataBytes, nil
}

// initPools always fetches all Hashflow pools
func (d *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	marketMarkers, err := d.client.ListMarketMakers(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch market makers")
		return []entity.Pool{}, err
	}

	pairs, err := d.client.ListPriceLevels(ctx, marketMarkers)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch price levels")
		return []entity.Pool{}, err
	}

	var newPools []entity.Pool
	for _, pair := range pairs {
		pool, err := d.createPool(pair)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Warn("failed to create pool. skip this pool")
			continue
		}
		newPools = append(newPools, pool)
	}

	logger.Infof("got total of %v Hashflow pools", len(newPools))

	return newPools, nil
}

func (d *PoolsListUpdater) createPool(pair Pair) (entity.Pool, error) {
	poolID := PoolID{MarketMaker: pair.MarketMaker, Token0: pair.Tokens[0], Token1: pair.Tokens[1]}
	poolToken0 := &entity.PoolToken{
		Address:   pair.Tokens[0],
		Decimals:  pair.Decimals[0],
		Swappable: true,
	}
	poolToken1 := &entity.PoolToken{
		Address:   pair.Tokens[1],
		Decimals:  pair.Decimals[1],
		Swappable: true,
	}

	staticExtra := StaticExtra{
		MarketMaker: pair.MarketMaker,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err, "poolID": poolID.String()}).
			Error("create pool failed > marshal static extra failed")
		return entity.Pool{}, err
	}

	extra := Extra{
		OneToZeroPriceLevels: pair.OneToZeroPriceLevels,
		ZeroToOnePriceLevels: pair.ZeroToOnePriceLevels,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err, "poolID": poolID.String()}).
			Error("create pool failed > marshal extra failed")
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     poolID.String(),
		Exchange:    d.config.DexID,
		Type:        DexTypeHashflow,
		Tokens:      []*entity.PoolToken{poolToken0, poolToken1},
		Reserves:    calcReserves(pair),
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		Timestamp:   time.Now().Unix(),
	}, nil
}
