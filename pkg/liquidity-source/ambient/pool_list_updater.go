package ambient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolListUpdater struct {
	cfg      Config
	subgraph *graphql.Client
}

func NewPoolsListUpdater(
	cfg Config,
) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:      cfg,
		subgraph: graphqlPkg.NewWithTimeout(cfg.SubgraphURL, cfg.SubgraphRequestTimeout.Duration),
	}
}

// GetNewPools: fetch new pools from the subgraph
func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.cfg.DexID
		startTime = time.Now()
		meta      PoolListUpdaterMetadata
	)
	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	if err := json.Unmarshal(metadataBytes, &meta); err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("unmarshal metadata failed")
		return nil, nil, err
	}

	sPools, err := u.fetchSubgraph(ctx, meta.LastCreateTime)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("fetchSubgraph failed")
		return nil, metadataBytes, err
	}
	if len(sPools) == 0 {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":    dexID,
					"pools_len": 0,
					// "last_created_time": lastCreateTime,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pools")
		return nil, metadataBytes, nil
	}

	entPools, err := u.toEntPools(sPools)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("transform to entity pool failed")
		return nil, nil, err
	}

	newMetaBytes, err := u.metadataBytes(sPools)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("get new metadata bytes failed")
		return nil, nil, err
	}

	return entPools, newMetaBytes, nil
}

func (u *PoolListUpdater) fetchSubgraph(ctx context.Context, lastCreateTime uint64) ([]Pool, error) {
	var (
		req = graphql.NewRequest(fmt.Sprintf(`{
	pools(
		where: {
			timeCreate_gt: %d,
		}
		orderBy: timeCreate,
		orderDirection: desc,
		first: %d,
	) {
		id
		blockCreate
		timeCreate
		base
		quote
		poolIdx
	}
}`, lastCreateTime, fetchLimit))
		resp FetchPoolsResponse
	)

	if err := u.subgraph.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Pools, nil
}

func (u *PoolListUpdater) toEntPools(subgraphPools []Pool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))

	for _, sPool := range subgraphPools {
		// each ambient virtual pool only have 2 tokens (base & quote)
		tokens := []*entity.PoolToken{
			{
				Address:   sPool.Base,
				Swappable: true,
			},
			{
				Address:   sPool.Quote,
				Swappable: true,
			},
		}
		reserves := []string{"0", "0"}

		sExtra := StaticExtra{
			Base:    sPool.Base,
			Quote:   sPool.Quote,
			PoolIdx: sPool.PoolIdx,
		}
		sExtraBytes, err := json.Marshal(sExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     sPool.ID,
			Exchange:    u.cfg.DexID,
			Type:        DexTypeAmbient,
			Tokens:      tokens,
			Reserves:    reserves,
			StaticExtra: string(sExtraBytes),
		})
	}

	return pools, nil
}

func (u *PoolListUpdater) metadataBytes(subgraphPools []Pool) ([]byte, error) {
	var metadata PoolListUpdaterMetadata
	if len(subgraphPools) == 0 {
		metadata.LastCreateTime = 0
	} else {
		metadata.LastCreateTime = subgraphPools[len(subgraphPools)-1].TimeCreate
	}

	return json.Marshal(metadata)
}
