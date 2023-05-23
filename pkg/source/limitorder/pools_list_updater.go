package limitorder

import (
	"context"
	"strings"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config           *Config
	limitOrderClient *httpClient
}

func NewPoolsListUpdater(
	cfg *Config,
) (*PoolsListUpdater, error) {
	limitOrderClient := NewHTTPClient(cfg.LimitOrderHTTPUrl)
	return &PoolsListUpdater{
		config:           cfg,
		limitOrderClient: limitOrderClient,
	}, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	loPairs, err := d.limitOrderClient.ListAllPairs(ctx, ChainID(d.config.ChainID))
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("can not get list all pairs")
		return nil, metadataBytes, err
	}
	if len(loPairs) == 0 {
		return nil, metadataBytes, nil
	}

	pairs := d.extractTokenPairs(loPairs)
	pools := make([]entity.Pool, len(pairs))
	for i, pair := range pairs {
		newPool := d.initPool(pair)
		pools[i] = newPool
	}

	if len(pools) > 0 {
		logger.Infof("[LimitOrder] got total %v pools", len(pools))
	}

	return pools, metadataBytes, nil
}

func (d *PoolsListUpdater) extractTokenPairs(loPairs []*limitOrderPair) []*tokenPair {
	pairMap := make(map[string]*tokenPair, 0)
	for _, loPair := range loPairs {
		pair := d.toTokenPair(loPair)
		poolID := d.getPoolID(pair.Token0, pair.Token1)
		if _, ok := pairMap[poolID]; !ok {
			pairMap[poolID] = pair
		}
	}
	result := make([]*tokenPair, 0, len(pairMap))
	for _, pair := range pairMap {
		result = append(result, pair)
	}

	return result
}

func (d *PoolsListUpdater) toTokenPair(pair *limitOrderPair) *tokenPair {
	token0, token1 := strings.ToLower(pair.MakerAsset), strings.ToLower(pair.TakeAsset)
	if token0 > token1 {
		return &tokenPair{
			Token0: token0,
			Token1: token1,
		}
	}
	return &tokenPair{
		Token0: token1,
		Token1: token0,
	}
}

func (d *PoolsListUpdater) getPoolID(token0, token1 string) string {
	token0, token1 = strings.ToLower(token0), strings.ToLower(token1)
	if token0 > token1 {
		return strings.Join([]string{PrefixLimitOrderPoolID, token0, token1}, SeparationCharacterLimitOrderPoolID)
	}
	return strings.Join([]string{PrefixLimitOrderPoolID, token1, token0}, SeparationCharacterLimitOrderPoolID)
}

func (d *PoolsListUpdater) initPool(pair *tokenPair) entity.Pool {
	newPool := entity.Pool{
		Address:  d.getPoolID(pair.Token0, pair.Token1),
		Exchange: d.config.DexID,
		Type:     DexTypeLimitOrder,
		Reserves: entity.PoolReserves{limitOrderPoolReserve, limitOrderPoolReserve},
	}
	if strings.ToLower(pair.Token0) > strings.ToLower(pair.Token1) {
		newPool.Tokens = []*entity.PoolToken{
			{
				Address:   pair.Token0,
				Swappable: true,
			},
			{
				Address:   pair.Token1,
				Swappable: true,
			},
		}
	} else {
		newPool.Tokens = []*entity.PoolToken{
			{
				Address:   pair.Token1,
				Swappable: true,
			},
			{
				Address:   pair.Token0,
				Swappable: true,
			},
		}
	}

	return newPool
}
