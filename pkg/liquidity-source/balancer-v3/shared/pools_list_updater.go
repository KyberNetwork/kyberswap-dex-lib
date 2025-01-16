package shared

import (
	"context"
	"math/big"
	"net/http"

	"github.com/goccy/go-json"

	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		graphqlClient *graphqlpkg.Client
	}

	Config struct {
		DexID           string
		SubgraphAPI     string
		SubgraphHeaders http.Header
		NewPoolLimit    int
		Factory         string
	}

	Metadata struct {
		LastBlockTimestamp *big.Int `json:"lastBlockTimestamp"`
	}
)

func NewPoolsListUpdater(config *Config, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
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
	if metadata.LastBlockTimestamp == nil {
		metadata.LastBlockTimestamp = big.NewInt(0)
	}

	pools, lastBlockTimestamp, err := u.querySubgraph(ctx, metadata.LastBlockTimestamp)
	if err != nil {
		return nil, nil, err
	}

	if len(pools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastBlockTimestamp: lastBlockTimestamp,
	})
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) querySubgraph(ctx context.Context, lastBlockTimestamp *big.Int) ([]*SubgraphPool, *big.Int, error) {
	var response struct {
		Pools []*SubgraphPool `json:"pools"`
	}

	query := BuildSubgraphPoolsQuery(
		u.config.Factory,
		lastBlockTimestamp,
		u.config.NewPoolLimit,
		0,
	)

	req := graphqlpkg.NewRequest(query)
	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, nil, err
	}

	if len(response.Pools) != 0 {
		lastBlockTimestamp, _ = new(big.Int).SetString(response.Pools[len(response.Pools)-1].BlockTimestamp, 10)
	}

	return response.Pools, lastBlockTimestamp, nil
}
