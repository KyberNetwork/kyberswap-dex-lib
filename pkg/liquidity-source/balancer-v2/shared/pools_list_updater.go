package shared

import (
	"context"
	"math/big"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/machinebox/graphql"

	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		graphqlClient *graphql.Client
	}

	Config struct {
		DexID           string
		SubgraphAPI     string
		SubgraphHeaders http.Header
		NewPoolLimit    int
		PoolTypes       []string
	}

	Metadata struct {
		LastCreateTime *big.Int `json:"lastCreateTime"`
	}
)

const graphQLRequestTimeout = 20 * time.Second

func NewPoolsListUpdater(config *Config) *PoolsListUpdater {
	graphqlClient := graphqlpkg.New(graphqlpkg.Config{
		Url:     config.SubgraphAPI,
		Header:  config.SubgraphHeaders,
		Timeout: graphQLRequestTimeout,
	})

	return &PoolsListUpdater{
		config:        config,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]*SubgraphPool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}
	if metadata.LastCreateTime == nil {
		metadata.LastCreateTime = big.NewInt(0)
	}

	pools, lastCreateTime, err := u.querySubgraph(ctx, metadata.LastCreateTime)
	if err != nil {
		return nil, nil, err
	}

	if len(pools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreateTime: lastCreateTime,
	})
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) querySubgraph(ctx context.Context, lastCreateTime *big.Int) ([]*SubgraphPool, *big.Int, error) {
	var response struct {
		Pools []*SubgraphPool `json:"pools"`
	}

	query := BuildSubgraphPoolsQuery(
		u.config.PoolTypes,
		lastCreateTime,
		u.config.NewPoolLimit,
		0,
	)
	req := graphql.NewRequest(query)

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, nil, err
	}

	if len(response.Pools) != 0 {
		lastCreateTime = response.Pools[len(response.Pools)-1].CreateTime
	}

	return response.Pools, lastCreateTime, nil
}
