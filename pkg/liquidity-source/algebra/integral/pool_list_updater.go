package integral

import (
	"context"
	"fmt"

	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"

	"math/big"
	"strconv"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphqlpkg.Client
}

func NewPoolsListUpdater(
	cfg *Config,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, lastPoolIds []string, first, skip int) ([]SubgraphPool, error) {
	allowSubgraphError := d.config.AllowSubgraphError

	req := graphqlpkg.NewRequest(getPoolsListQuery(allowSubgraphError, lastCreatedAtTimestamp, lastPoolIds, first, skip))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err, _ := d.graphqlClient.Run(ctx, req, &response); err != nil {
		if allowSubgraphError && len(response.Pools) > 0 {
			return response.Pools, nil
		}

		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to query subgraph")
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

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, metadata.LastPoolIds, graphFirstLimit, 0)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pools list from subgraph")
		return nil, metadataBytes, err
	}

	numSubgraphPools := len(subgraphPools)

	logger.Infof("got %v subgraph pools from %v subgraph", numSubgraphPools, d.config.DexID)

	if numSubgraphPools == 0 {
		// no new pool
		return []entity.Pool{}, metadataBytes, nil
	}

	// Track the last pool's CreatedAtTimestamp
	lastPoolIds := []string{}
	lastCreatedAtTimestampStr := subgraphPools[numSubgraphPools-1].CreatedAtTimestamp
	lastCreatedAtTimestamp, ok := new(big.Int).SetString(lastCreatedAtTimestampStr, 10)
	if !ok {
		return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v",
			lastCreatedAtTimestampStr, subgraphPools[numSubgraphPools-1].ID)
	}

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		tokens := make([]*entity.PoolToken, 0, 2)
		reserves := make([]string, 0, 2)

		if p.Token0.Address != emptyString {
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
			reserves = append(reserves, zeroString)
		}

		if p.Token1.Address != emptyString {
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
			reserves = append(reserves, zeroString)
		}

		staticExtra, err := json.Marshal(&StaticExtra{
			UseBasePluginV2: d.config.UseBasePluginV2,
		})
		if err != nil {
			return nil, metadataBytes, err
		}

		createdAtTimestamp, err := kutils.Atoi[int64](p.CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v", p.CreatedAtTimestamp, p.ID)
		}

		var newPool = entity.Pool{
			Address:      p.ID,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			Exchange:     d.config.DexID,
			Type:         DexType,
			Timestamp:    createdAtTimestamp,
			Reserves:     reserves,
			Tokens:       tokens,
			StaticExtra:  string(staticExtra),
		}

		pools = append(pools, newPool)
		if p.CreatedAtTimestamp == lastCreatedAtTimestampStr {
			lastPoolIds = append(lastPoolIds, p.ID)
		}
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
		LastPoolIds:            lastPoolIds,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("got %v %v pools", len(pools), d.config.DexID)

	return pools, newMetadataBytes, nil
}
