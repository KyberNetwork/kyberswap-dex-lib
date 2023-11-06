package kyberpmm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config Config
	client IClient
}

func NewPoolsListUpdater(cfg Config, client IClient) *PoolsListUpdater {
	return &PoolsListUpdater{
		config: cfg,
		client: client,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	tokens, err := u.client.ListTokens(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("can not list all tokens")
		return nil, metadataBytes, err
	}

	pmmPairs, err := u.client.ListPairs(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("can not list all pmmPairs")
		return nil, metadataBytes, err
	}

	if len(pmmPairs) == 0 {
		return nil, metadataBytes, nil
	}

	pools := u.extractPMMPairs(tokens, pmmPairs)

	if len(pools) > 0 {
		logger.Infof("[Kyber PMM] got total %v pools", len(pools))
	}

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) extractPMMPairs(
	tokens map[string]TokenItem,
	pmmPairs map[string]PairItem,
) []entity.Pool {
	result := make([]entity.Pool, 0, len(pmmPairs))
	for pairID, pmmPair := range pmmPairs {
		newPool, err := u.transformToPool(tokens, pairID, pmmPair)
		if err != nil {
			logger.Errorf("failed to convert %v to token pair, err: %v", pmmPair, err)
			continue
		}

		result = append(result, newPool)
	}

	return result
}

func (u *PoolsListUpdater) transformToPool(
	tokens map[string]TokenItem,
	pairID string,
	pairItemResponse PairItem,
) (entity.Pool, error) {
	baseToken, ok := tokens[pairItemResponse.Base]
	if !ok {
		return entity.Pool{}, ErrTokenNotFound
	}

	quoteToken, ok := tokens[pairItemResponse.Quote]
	if !ok {
		return entity.Pool{}, ErrTokenNotFound
	}

	staticExtra := StaticExtra{
		PairID:            pairID,
		BaseTokenAddress:  baseToken.Address,
		QuoteTokenAddress: quoteToken.Address,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshal static extra data")
		return entity.Pool{}, err
	}

	newPool := entity.Pool{
		Address:   u.getPoolID(baseToken.Address, quoteToken.Address),
		Exchange:  u.config.DexID,
		Type:      DexTypeKyberPMM,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{poolReserve, poolReserve},
		Tokens: []*entity.PoolToken{
			{
				Address:   strings.ToLower(baseToken.Address),
				Decimals:  baseToken.Decimals,
				Swappable: true,
				Symbol:    baseToken.Symbol,
				Name:      baseToken.Name,
			},
			{
				Address:   strings.ToLower(quoteToken.Address),
				Decimals:  quoteToken.Decimals,
				Swappable: true,
				Symbol:    quoteToken.Symbol,
				Name:      quoteToken.Name,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	return newPool, nil
}

func (u *PoolsListUpdater) getPoolID(token0, token1 string) string {
	token0, token1 = strings.ToLower(token0), strings.ToLower(token1)
	if token0 < token1 {
		return strings.Join([]string{PoolIDPrefix, token0, token1}, PoolIDSeparator)
	}

	return strings.Join([]string{PoolIDPrefix, token1, token0}, PoolIDSeparator)
}
