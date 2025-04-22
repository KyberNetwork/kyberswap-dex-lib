package shared

import (
	"context"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/goccy/go-json"

	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		graphqlClient *graphqlpkg.Client
	}

	Metadata struct {
		LastBlockTimestamp int64 `json:"lastBlockTimestamp"`
		Skip               int   `json:"skip"`
	}
)

func NewPoolsListUpdater(config *Config, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]*SubgraphPool, []byte, error) {
	l := klog.WithFields(ctx, klog.Fields{
		"dexID": u.config.DexID,
	})
	l.Infof("Start updating pools list ...")

	var pools []*SubgraphPool
	defer func() {
		l.WithFields(klog.Fields{
			"count": len(pools),
		}).Infof("Finish updating pools list.")
	}()

	var meta Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, nil, err
		}
	}

	var err error
	if u.config.UseSubgraphV1 {
		pools, meta, err = u.getNewPoolsV1(ctx, meta)
	} else {
		pools, meta, err = u.getNewPoolsV2(ctx, meta)
	}

	if err != nil {
		return nil, metadataBytes, err
	} else if len(pools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetaBytes, err
}

func (u *PoolsListUpdater) getNewPoolsV1(ctx context.Context, metadata Metadata) ([]*SubgraphPool, Metadata, error) {
	var response struct {
		Pools []*SubgraphPoolV1 `json:"pools"`
	}

	query := BuildSubgraphPoolsQueryV1(
		u.config.SubgraphPoolTypes,
		metadata.LastBlockTimestamp,
		u.config.NewPoolLimit,
		metadata.Skip,
	)

	req := graphqlpkg.NewRequest(query)

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, metadata, err
	}

	if poolCnt := len(response.Pools); poolCnt >= u.config.NewPoolLimit {
		metadata.Skip += u.config.NewPoolLimit
	} else if poolCnt > 0 {
		metadata.LastBlockTimestamp = response.Pools[len(response.Pools)-1].CreateTime
		metadata.Skip = 0
	}

	return transformPools(response.Pools), metadata, nil
}

func (u *PoolsListUpdater) getNewPoolsV2(ctx context.Context, metadata Metadata) ([]*SubgraphPool, Metadata, error) {
	var response struct {
		Pools []*SubgraphPool `json:"poolGetPools"`
	}

	query := SubgraphPoolsQueryV2
	if u.config.SubgraphChain == "BERACHAIN" {
		query = SubgraphPoolsQueryBerachain
	}

	req := graphqlpkg.NewRequest(query)
	req.Var(VarChain, u.config.SubgraphChain)
	req.Var(VarPoolType, u.config.SubgraphPoolTypes)
	req.Var(VarCreateTimeGt, metadata.LastBlockTimestamp)
	req.Var(VarFirst, u.config.NewPoolLimit)
	req.Var(VarSkip, metadata.Skip)

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, metadata, err
	}

	if poolCnt := len(response.Pools); poolCnt >= u.config.NewPoolLimit {
		metadata.Skip += u.config.NewPoolLimit
	} else if poolCnt > 0 {
		metadata.LastBlockTimestamp = response.Pools[len(response.Pools)-1].CreateTime
		metadata.Skip = 0
	}

	return response.Pools, metadata, nil
}

func transformPools(poolsV1 []*SubgraphPoolV1) []*SubgraphPool {
	var result = make([]*SubgraphPool, 0, len(poolsV1))
	for _, p := range poolsV1 {
		newPool := &SubgraphPool{
			ID:         p.ID,
			Address:    p.Address,
			Type:       p.PoolType,
			Version:    p.PoolTypeVersion,
			CreateTime: p.CreateTime,
		}

		for _, token := range p.Tokens {
			newToken := PoolToken{
				Address:   token.Address,
				IsAllowed: true,
				Weight:    token.Weight,
				Decimals:  token.Decimals,
			}
			newPool.PoolTokens = append(newPool.PoolTokens, newToken)
		}

		result = append(result, newPool)
	}

	return result
}
