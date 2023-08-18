package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphql.Client
}

func NewPoolsListUpdater(
	cfg *Config,
) *PoolsListUpdater {
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)
	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, first, skip int) ([]SubgraphPool, error) {
	req := graphql.NewRequest(fmt.Sprintf(`{
		pools(where : {createdAtTimestamp_gte: %v}, first: %v, skip: %v, orderBy: createdAtTimestamp, orderDirection: asc) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			feeTier
			token0 {
				id
				name
				symbol
			  	decimals
			}
			token1 {
				id
				name
				symbol
			  	decimals
			}
		}
	}`, lastCreatedAtTimestamp, first, skip),
	)

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.Errorf("failed to query subgraph, err: %v", err)
		return nil, err
	}

	return response.Pools, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAtTimestamp: integer.Zero(),
	}
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, graphFirstLimit, 0)
	if err != nil {
		logger.Errorf("failed to get pools list from subgraph, err: %v", err)
		return nil, metadataBytes, err
	}

	numPools := len(subgraphPools)

	// Return immediately if number of new pools is 0
	if numPools == 0 {
		return nil, metadataBytes, nil
	}

	logger.Infof("got %v subgraph pools from Elastic subgraph", numPools)

	pools := make([]entity.Pool, 0, len(subgraphPools))

	for _, p := range subgraphPools {
		var tokens = make([]*entity.PoolToken, 0, 2)
		var reserves = make([]string, 0, 2)
		var staticField = StaticExtra{
			PoolId: p.ID,
		}

		if p.Token0.Address != "" {
			token0Decimals, err := strconv.Atoi(p.Token0.Decimals)

			if err != nil {
				token0Decimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   p.Token0.Address,
				Name:      p.Token0.Name,
				Symbol:    p.Token0.Symbol,
				Decimals:  uint8(token0Decimals),
				Weight:    defaultTokenWeight,
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, reserveZero)
		} else {
			logger.WithFields(logger.Fields{
				"tokenAddress": p.Token0.Address,
			}).Warn("subgraph pool's token address is empty")
		}

		if p.Token1.Address != "" {
			token1Decimals, err := strconv.Atoi(p.Token1.Decimals)

			if err != nil {
				token1Decimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   p.Token1.Address,
				Name:      p.Token1.Name,
				Symbol:    p.Token1.Symbol,
				Decimals:  uint8(token1Decimals),
				Weight:    defaultTokenWeight,
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, reserveZero)
		} else {
			logger.WithFields(logger.Fields{
				"tokenAddress": p.Token1.Address,
			}).Warn("subgraph pool's token address is empty")
		}

		var swapFee, _ = strconv.ParseFloat(p.FeeTier, 64)

		staticBytes, _ := json.Marshal(staticField)
		var newPool = entity.Pool{
			Address:      p.ID,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      swapFee,
			Exchange:     d.config.DexID,
			Type:         DexTypeElastic,
			Timestamp:    time.Now().Unix(),
			Reserves:     reserves,
			Tokens:       tokens,
			StaticExtra:  string(staticBytes),
		}

		pools = append(pools, newPool)
	}

	// Track the last pool's CreatedAtTimestamp
	var lastCreatedAtTimestamp = metadata.LastCreatedAtTimestamp
	if len(subgraphPools) > 0 {
		lastSubgraphPool := subgraphPools[len(subgraphPools)-1]
		ts, ok := new(big.Int).SetString(lastSubgraphPool.CreatedAtTimestamp, 10)
		if !ok {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v",
				lastSubgraphPool.CreatedAtTimestamp, lastSubgraphPool.ID)
		}

		lastCreatedAtTimestamp = ts
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("got %v Elastic pools", len(pools))

	return pools, newMetadataBytes, nil
}
