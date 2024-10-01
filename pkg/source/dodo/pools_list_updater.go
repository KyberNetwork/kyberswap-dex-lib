package dodo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphql.Client
}

func NewPoolsListUpdater(
	cfg *Config,
) *PoolsListUpdater {
	graphqlClient := graphqlpkg.New(graphqlpkg.Config{
		Url:     cfg.SubgraphAPI,
		Header:  cfg.SubgraphHeaders,
		Timeout: defaultGraphQLRequestTimeout,
	})

	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata = make(Metadata, len(subgraphTypeToPoolTypeMap))
	if len(metadataBytes) > 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			logger.WithFields(logger.Fields{
				"metadataBytes": metadataBytes,
				"error":         err,
			}).Errorf("failed to marshal metadata")
			return nil, metadataBytes, err
		}
	}

	var pools = make([]entity.Pool, 0, len(subgraphTypeToPoolTypeMap))
	for subgraphPoolType, poolType := range subgraphTypeToPoolTypeMap {
		var poolTypeMetadata PoolTypeMetadata
		var ok bool
		poolTypeMetadata, ok = metadata[poolType]
		if !ok {
			poolTypeMetadata.LastCreatedAtTimestamp = zeroBI
		}

		newPools, newPoolTypeMetadata, err := d.getNewPoolByType(ctx, poolType, subgraphPoolType, poolTypeMetadata)
		if err != nil {
			return nil, metadataBytes, err
		}

		metadata[poolType] = newPoolTypeMetadata
		pools = append(pools, newPools...)
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.WithFields(logger.Fields{
			"metadata": metadata,
			"error":    err,
		}).Errorf("failed to marshal metadata")
		return nil, metadataBytes, err
	}

	logger.Infof("got total %v Dodo pools", len(pools))

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getNewPoolByType(
	ctx context.Context,
	poolType string,
	subgraphPoolType string,
	metadata PoolTypeMetadata,
) ([]entity.Pool, PoolTypeMetadata, error) {
	subgraphPools, err := d.getPoolsList(ctx, d.config.NewPoolLimit, 0, subgraphPoolType, metadata.LastCreatedAtTimestamp)
	if err != nil {
		return nil, PoolTypeMetadata{}, err
	}

	logger.Infof("got %v pools from subgraph for type %v", len(subgraphPools), subgraphPoolType)

	pools := make([]entity.Pool, 0, len(subgraphPools))
	var staticExtra StaticExtra
	for _, pool := range subgraphPools {
		var tokens []*entity.PoolToken
		var reserves []string
		staticExtra = StaticExtra{
			PoolId:           pool.ID,
			LpToken:          pool.BaseLpToken.Address,
			Type:             pool.Type,
			Tokens:           []string{pool.BaseToken.Address, pool.QuoteToken.Address},
			DodoV1SellHelper: d.config.DodoV1SellHelper,
		}

		if pool.BaseToken.Address != "" {
			baseTokenDecimals, err := strconv.Atoi(pool.BaseToken.Decimals)
			if err != nil {
				logger.WithFields(logger.Fields{
					"decimals": pool.BaseToken.Decimals,
				}).Warn("failed to convert decimals from string to int")
				baseTokenDecimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   pool.BaseToken.Address,
				Name:      pool.BaseToken.Name,
				Symbol:    pool.BaseToken.Symbol,
				Decimals:  uint8(baseTokenDecimals),
				Weight:    defaultTokenWeight,
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, zeroString)
		} else {
			logger.WithFields(logger.Fields{
				"poolID": pool.ID,
			}).Errorf("base token address is empty")
		}

		if pool.QuoteToken.Address != "" {
			quoteTokenDecimals, err := strconv.Atoi(pool.QuoteToken.Decimals)
			if err != nil {
				logger.WithFields(logger.Fields{
					"decimals": pool.BaseToken.Decimals,
				}).Warn("failed to convert decimals from string to int")
				quoteTokenDecimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   pool.QuoteToken.Address,
				Name:      pool.QuoteToken.Name,
				Symbol:    pool.QuoteToken.Symbol,
				Decimals:  uint8(quoteTokenDecimals),
				Weight:    defaultTokenWeight,
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, zeroString)
		} else {
			logger.WithFields(logger.Fields{
				"poolID": pool.ID,
			}).Errorf("quote token address is empty")
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to marshal static extra data")
			return nil, PoolTypeMetadata{}, err
		}

		createdAtTimestamp, err := strconv.ParseInt(pool.CreatedAtTimestamp, 10, 64)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolID":             pool.ID,
				"createdAtTimestamp": pool.CreatedAtTimestamp,
			}).Errorf("failed to convert createdAtTimestamp from string to int")

			createdAtTimestamp = 0
		}

		var newPool = entity.Pool{
			Address:     pool.ID,
			Exchange:    d.config.DexID,
			Type:        poolType,
			Timestamp:   createdAtTimestamp,
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, newPool)
	}

	var lastCreatedAtTimestamp = metadata.LastCreatedAtTimestamp
	if len(subgraphPools) > 0 {
		lastSubgraphPool := subgraphPools[len(subgraphPools)-1]
		ts, ok := new(big.Int).SetString(lastSubgraphPool.CreatedAtTimestamp, 10)
		if !ok {
			logger.WithFields(logger.Fields{
				"createdAtTimestamp": lastSubgraphPool.CreatedAtTimestamp,
				"poolID":             lastSubgraphPool.ID,
				"error":              err,
			}).Errorf("failed to set string createdAtTimestamp to *big.Int")

			return nil, PoolTypeMetadata{}, err
		}

		lastCreatedAtTimestamp = ts
	}
	logger.Infof("got %v Dodo pools for type %v", len(pools), poolType)

	return pools, PoolTypeMetadata{LastCreatedAtTimestamp: lastCreatedAtTimestamp}, nil
}

func (d *PoolsListUpdater) getPoolsList(
	ctx context.Context,
	first, skip int,
	dexType string,
	lastCreateTime *big.Int,
) ([]SubgraphPool, error) {
	// 'CLASSICAL', 'DVM', 'DSP', 'DPP' pools
	req := graphql.NewRequest(fmt.Sprintf(`{
		pairs(
				first: %v, 
				skip: %v, 
				where: {
						type: "%v"
						createdAtTimestamp_gte: %v
				}, 
				orderBy: createdAtTimestamp, 
				orderDirection: asc
		){
			id
			baseToken {
			    id
			    name
			    symbol
			    decimals
			}
			quoteToken {
			    id
			    name
			    symbol
			    decimals
			}
			baseLpToken { # LP token
			    id
				name
				symbol
				decimals
			}
			i
			k
			mtFeeRate
			lpFeeRate
			baseReserve
			quoteReserve
			isTradeAllowed
			type
            createdAtTimestamp
		}
	}`, first, skip, dexType, lastCreateTime))

	var response struct {
		Pairs []SubgraphPool `json:"pairs"`
	}
	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.Errorf("failed to query subgraph, err %v", err)
		return nil, err
	}

	return response.Pairs, nil
}
