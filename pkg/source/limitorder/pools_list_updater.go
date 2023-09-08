package limitorder

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config           *Config
	limitOrderClient *httpClient

	includedContractAddresses mapset.Set[string]
}

func NewPoolsListUpdater(
	cfg *Config,
) (*PoolsListUpdater, error) {
	limitOrderClient := NewHTTPClient(cfg.LimitOrderHTTPUrl)
	contractAddresses := lo.Map(cfg.ContractAddresses, func(c string, _ int) string { return strings.ToLower(c) })
	return &PoolsListUpdater{
		config:           cfg,
		limitOrderClient: limitOrderClient,

		includedContractAddresses: mapset.NewSet(contractAddresses...),
	}, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	loPairs, err := d.limitOrderClient.ListAllPairs(ctx, ChainID(d.config.ChainID), d.config.SupportMultiSCs)
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
	pools := make([]entity.Pool, 0, len(pairs))
	for _, pair := range pairs {
		newPool, err := d.initPool(pair)
		if err != nil {
			continue
		}
		pools = append(pools, newPool)
	}

	if len(pools) > 0 {
		logger.Infof("[LimitOrder] got total %v pools", len(pools))
	}

	return pools, metadataBytes, nil
}

func (d *PoolsListUpdater) extractTokenPairs(loPairs []*limitOrderPair) []*tokenPair {
	pairMap := make(map[string]*tokenPair, 0)
	for _, loPair := range loPairs {
		// if we're filtering by address, and if this pool SC doesn't match, then ignore
		if d.includedContractAddresses.Cardinality() > 0 && !d.includedContractAddresses.Contains(strings.ToLower(loPair.ContractAddress)) {
			continue
		}
		pair := d.toTokenPair(loPair)
		poolID := d.getPoolID(pair.Token0, pair.Token1, pair.ContractAddress)
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
			Token0:          token0,
			Token1:          token1,
			ContractAddress: strings.ToLower(pair.ContractAddress),
		}
	}
	return &tokenPair{
		Token0:          token1,
		Token1:          token0,
		ContractAddress: strings.ToLower(pair.ContractAddress),
	}
}

func (d *PoolsListUpdater) getPoolID(token0, token1, contractAddress string) string {
	token0, token1 = strings.ToLower(token0), strings.ToLower(token1)
	var poolId string
	if token0 > token1 {
		poolId = strings.Join([]string{PrefixLimitOrderPoolID, token0, token1}, SeparationCharacterLimitOrderPoolID)
	} else {
		poolId = strings.Join([]string{PrefixLimitOrderPoolID, token1, token0}, SeparationCharacterLimitOrderPoolID)
	}
	if d.config.SupportMultiSCs {
		return strings.Join([]string{poolId, contractAddress}, SeparationCharacterLimitOrderPoolID)
	}
	return poolId
}

func (d *PoolsListUpdater) initPool(pair *tokenPair) (entity.Pool, error) {
	staticExtra := StaticExtra{ContractAddress: pair.ContractAddress}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"token0":          pair.Token0,
			"token1":          pair.Token1,
			"contractAddress": pair.ContractAddress,
			"error":           err,
		}).Errorf("failed to marshal static extra data")
		return entity.Pool{}, err
	}

	newPool := entity.Pool{
		Address:  d.getPoolID(pair.Token0, pair.Token1, pair.ContractAddress),
		Exchange: d.config.DexID,
		Type:     DexTypeLimitOrder,
		Reserves: entity.PoolReserves{limitOrderPoolReserve, limitOrderPoolReserve},

		StaticExtra: string(staticExtraBytes),
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

	return newPool, nil
}
