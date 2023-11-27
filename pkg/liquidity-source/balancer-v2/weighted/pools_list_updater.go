package weighted

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/subgraph"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	graphqlClient := graphqlpkg.NewWithTimeout(config.SubgraphAPI, graphQLRequestTimeout)

	return &PoolsListUpdater{
		config:        *config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId":   u.config.DexID,
		"dexType": DexTypeBalancerWeighted,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexTypeBalancerWeighted,
		}).Infof("Finish updating pools list.")
	}()

	var metadata Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexTypeBalancerWeighted,
		}).Error(err.Error())

		return nil, nil, err
	}

	subgraphPools, lastCreateTime, err := u.querySubgraph(ctx, metadata.LastCreateTime)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexTypeBalancerWeighted,
		}).Error(err.Error())

		return nil, nil, err
	}

	if len(subgraphPools) == 0 {
		return nil, nil, nil
	}

	pools, err := u.initPools(ctx, subgraphPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexTypeBalancerWeighted,
		}).Error(err.Error())

		return nil, nil, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreateTime: lastCreateTime,
	})
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, subgraphPools []*subgraph.Pool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, len(subgraphPools))

	for i, subgraphPool := range subgraphPools {
		poolTokens := make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves := make([]string, len(subgraphPool.Tokens))
		for j, token := range subgraphPool.Tokens {
			reserves[j] = "0"

			w, err := strconv.ParseFloat(token.Weight, 64)
			if err != nil {
				return nil, err
			}
			weight := uint(w * 1e18)
			if weight == 0 {
				weight = uint(1e18 / len(subgraphPool.Tokens))
			}

			poolTokens[j] = &entity.PoolToken{
				Address:   token.Address,
				Weight:    weight,
				Swappable: true,
			}
		}

		staticExtra := StaticExtra{
			PoolID:          subgraphPool.ID,
			PoolTypeVersion: subgraphPool.PoolTypeVersion,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     subgraphPool.Address,
			Exchange:    u.config.DexID,
			Type:        DexTypeBalancerWeighted,
			Timestamp:   time.Now().Unix(),
			Tokens:      poolTokens,
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
		}

	}

	return pools, nil
}

func (u *PoolsListUpdater) querySubgraph(ctx context.Context, lastCreateTime *big.Int) ([]*subgraph.Pool, *big.Int, error) {
	var response struct {
		Pools []*subgraph.Pool `json:"pools"`
	}

	query := subgraph.GetPoolsQuery(
		poolTypeWeighted,
		lastCreateTime,
		u.config.NewPoolLimit,
		0,
	)

	req := graphql.NewRequest(query)

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexTypeBalancerWeighted,
		}).Error(err.Error())

		return nil, nil, err
	}

	if len(response.Pools) != 0 {
		lastCreateTime = response.Pools[len(response.Pools)-1].CreateTime
	}

	return response.Pools, lastCreateTime, nil
}
