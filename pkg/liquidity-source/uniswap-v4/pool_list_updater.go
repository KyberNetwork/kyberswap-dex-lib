package uniswapv4

import (
	"context"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	PoolsListUpdaterMetadata struct {
		LastCreatedAtTimestamp int `json:"lastCreatedAtTimestamp"`
		LastProcessedPoolID    int `json:"lastProcessedPoolID"`
	}
)

func NewPoolListUpdater(
	config Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	// l := logger.WithFields(logger.Fields{"dex_id": u.config.DexID})
	return nil, nil, nil
}
