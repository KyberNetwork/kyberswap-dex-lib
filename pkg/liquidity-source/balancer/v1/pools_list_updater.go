package balancerv1

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	PoolsListUpdaterMetadata struct {
		LastCreateTime int `json:"lastCreateTime"`
	}

	FetchPoolsResponse struct {
		Pools []FetchPoolsResponsePool `json:"pools"`
	}

	FetchPoolsResponsePool struct {
		ID         string   `json:"id"`
		TokensList []string `json:"tokensList"`
		CreateTime int      `json:"createTime"`
	}
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	lastCreateTime, err := u.getLastCreateTime(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getLastCreateTime failed")
	}

	ctx = util.NewContextWithTimestamp(ctx)

	subgraphPools, err := u.fetchPoolsFromSubgraph(ctx, lastCreateTime)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("fetchPoolsFromSubgraph failed")
		return nil, metadataBytes, err
	}

	if len(subgraphPools) == 0 {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":            dexID,
					"pools_len":         0,
					"last_created_time": lastCreateTime,
					"duration_ms":       time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pools")
		return nil, metadataBytes, nil
	}

	pools, err := u.initPools(ctx, subgraphPools)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetaData, err := newPoolsListUpdaterMetadata(subgraphPools[len(subgraphPools)-1].CreateTime)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newPoolsListUpdaterMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":            dexID,
				"pools_len":         len(pools),
				"last_created_time": lastCreateTime,
				"duration_ms":       time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetaData, nil
}

func (u *PoolsListUpdater) initPools(_ context.Context, subgraphPools []FetchPoolsResponsePool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, subgraphPool := range subgraphPools {
		poolTokens := make([]*entity.PoolToken, 0, len(subgraphPool.TokensList))
		reserves := make([]string, 0, len(subgraphPool.TokensList))
		for _, tokenAddress := range subgraphPool.TokensList {
			poolTokens = append(poolTokens, &entity.PoolToken{Address: tokenAddress, Swappable: true})
			reserves = append(reserves, "0")
		}

		pools = append(pools, entity.Pool{
			Address:  subgraphPool.ID,
			Exchange: u.config.DexID,
			Type:     DexType,
			Tokens:   poolTokens,
			Reserves: reserves,
		})
	}

	return pools, nil
}

func (u *PoolsListUpdater) fetchPoolsFromSubgraph(ctx context.Context, lastCreateTime int) ([]FetchPoolsResponsePool, error) {
	var (
		req  = graphqlpkg.NewRequest(newFetchPoolIDsQuery(lastCreateTime, u.config.NewPoolLimit))
		resp FetchPoolsResponse
	)

	if err := u.graphqlClient.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Pools, nil
}

func (u *PoolsListUpdater) getLastCreateTime(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.LastCreateTime, nil
}

func newPoolsListUpdaterMetadata(lastCreateTime int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		LastCreateTime: lastCreateTime,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func newFetchPoolIDsQuery(lastCreateTime int, first int) string {
	return fmt.Sprintf(`{
		pools(
			where : {
				createTime_gt: %d,
			},
			first: %d,
			orderBy: createTime,
			orderDirection: asc,
		) {
			id
			tokensList
			createTime
		}
	}`, lastCreateTime, first)
}
