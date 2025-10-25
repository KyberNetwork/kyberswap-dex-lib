package shared

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
		count         int
	}

	Metadata struct {
		LastBlockTimestamp int64 `json:"lastBlockTimestamp"`
		Skip               int   `json:"skip"`
	}
)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	l := klog.WithFields(ctx, klog.Fields{
		"dexID": u.config.DexID,
	})
	l.Infof("Start updating pools list ...")
	var pools []entity.Pool
	defer func() {
		l.WithFields(klog.Fields{
			"count": len(pools),
		}).Infof("Finish updating pools list.")
	}()

	var metadata Metadata
	if len(metadataBytes) > 0 && u.count%RelistInterval > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}
	u.count++

	subgraphPools, metadata, err := u.querySubgraph(ctx, metadata)
	if err != nil {
		return nil, nil, err
	} else if len(subgraphPools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, err
	}

	pools, err = u.initPools(ctx, subgraphPools)
	return pools, newMetadataBytes, err
}

func (u *PoolsListUpdater) querySubgraph(ctx context.Context, metadata Metadata) ([]*SubgraphPool, Metadata, error) {
	var response struct {
		Pools []*SubgraphPool `json:"poolGetPools"`
	}

	req := graphqlpkg.NewRequest(SubgraphPoolsQuery)
	req.Var(VarChain, u.config.SubgraphChain)
	req.Var(VarPoolType, u.config.SubgraphPoolType)
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

func (u *PoolsListUpdater) initPools(ctx context.Context, subgraphPools []*SubgraphPool) ([]entity.Pool, error) {
	bufferSet := mapset.NewThreadUnsafeSet[string]()
	for _, subgraphPool := range subgraphPools {
		for _, token := range subgraphPool.PoolTokens {
			if isBuffer := token.CanUseBufferForSwaps &&
				!lo.ContainsBy(subgraphPool.PoolTokens, func(t SubgraphToken) bool {
					// don't use as buffer token if the underlying token is already contained in the pool as a main token
					return token.UnderlyingToken.Address == t.Address
				}); isBuffer {
				bufferSet.Add(token.Address)
			}
		}
	}
	buffers := bufferSet.ToSlice()
	bufferAssets := make([]common.Address, len(buffers))
	req := u.ethrpcClient.R().SetContext(ctx)
	for i, buffer := range buffers {
		req.AddCall(&ethrpc.Call{
			ABI:    VaultExplorerABI,
			Target: u.config.VaultExplorer,
			Method: VaultMethodGetBufferAsset,
			Params: []any{common.HexToAddress(buffer)},
		}, []any{&bufferAssets[i]})
	}
	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}
	for i, bufferAsset := range bufferAssets {
		if bufferAsset == (common.Address{}) {
			bufferSet.Remove(buffers[i])
		}
	}

	pools := make([]entity.Pool, len(subgraphPools))
	for i, subgraphPool := range subgraphPools {
		bufferTokens := make([]string, len(subgraphPool.PoolTokens))
		poolTokens := make([]*entity.PoolToken, len(subgraphPool.PoolTokens))
		reserves := make([]string, len(subgraphPool.PoolTokens))
		for j, token := range subgraphPool.PoolTokens {
			isBuffer := token.CanUseBufferForSwaps && bufferSet.ContainsOne(token.Address) &&
				!lo.ContainsBy(subgraphPool.PoolTokens, func(t SubgraphToken) bool {
					// don't use as buffer token if the underlying token is already contained in the pool as a main token
					return token.UnderlyingToken.Address == t.Address
				})
			bufferTokens[j] = lo.Ternary(isBuffer, token.Address, "")
			poolTokens[j] = &entity.PoolToken{
				Address:   lo.Ternary(isBuffer, token.UnderlyingToken.Address, token.Address),
				Swappable: true,
			}
			reserves[j] = "0"
		}
		for _, bufferToken := range bufferTokens {
			if bufferToken != "" {
				poolTokens = append(poolTokens, &entity.PoolToken{
					Address:   bufferToken,
					Swappable: true,
				})
				reserves = append(reserves, "0")
			}
		}

		staticExtraBytes, _ := json.Marshal(&StaticExtra{
			Hook:         subgraphPool.Hook.Address,
			HookType:     subgraphPool.Hook.Type,
			BufferTokens: bufferTokens,
		})

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
