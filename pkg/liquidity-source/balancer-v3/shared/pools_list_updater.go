package shared

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		graphqlClient *graphqlpkg.Client
	}

	Metadata struct {
		LastBlockTimestamp int64 `json:"lastBlockTimestamp"`
	}
)

func NewPoolsListUpdater(config *Config, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.config.Factory == "" {
		return nil, nil, ErrEmptyFactoryConfig
	}

	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	subgraphPools, lastBlockTimestamp, err := u.querySubgraph(ctx, metadata.LastBlockTimestamp)
	if err != nil {
		return nil, nil, err
	} else if len(subgraphPools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastBlockTimestamp: lastBlockTimestamp,
	})
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(subgraphPools)
	return pools, newMetadataBytes, err
}

func (u *PoolsListUpdater) querySubgraph(ctx context.Context, lastBlockTimestamp int64) ([]*SubgraphPool, int64,
	error) {
	var response struct {
		Pools []*SubgraphPool `json:"pools"`
	}

	req := graphqlpkg.NewRequest(SubgraphPoolsQuery)
	req.Var(VarFactory, u.config.Factory)
	req.Var(VarBlockTimestampGte, lastBlockTimestamp)
	req.Var(VarFirst, u.config.NewPoolLimit)

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, 0, err
	}

	if len(response.Pools) > 0 {
		lastBlockTimestamp = response.Pools[len(response.Pools)-1].BlockTimestamp
	}

	return response.Pools, lastBlockTimestamp, nil
}

func (u *PoolsListUpdater) initPools(subgraphPools []*SubgraphPool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, len(subgraphPools))
	for i, subgraphPool := range subgraphPools {
		staticExtraBytes, err := json.Marshal(&StaticExtra{
			Vault:             subgraphPool.Vault.ID,
			DefaultHook:       u.config.DefaultHook,
			IsPoolInitialized: subgraphPool.IsInitialized,
			BufferTokens: lo.Map(subgraphPool.Tokens, func(token SubgraphToken, i int) string {
				return lo.Ternary(token.Buffer == nil, "", token.Address)
			}),
		})
		if err != nil {
			return nil, err
		}

		poolTokens := make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves := make([]string, len(subgraphPool.Tokens))
		for j, token := range subgraphPool.Tokens {
			address := token.Address
			if token.Buffer != nil {
				address = token.Buffer.UnderlyingToken.ID
			}
			poolTokens[j] = &entity.PoolToken{
				Address:   address,
				Swappable: true,
			}
			reserves[j] = "0"
		}

		pools[i] = entity.Pool{
			Address:     subgraphPool.Address,
			Exchange:    u.config.DexID,
			Type:        u.config.PoolType,
			Timestamp:   time.Now().Unix(),
			Tokens:      poolTokens,
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}
