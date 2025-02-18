package uniswapv4

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type (
	PoolsListUpdater struct {
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	Metadata struct {
		LastCreatedAtTimestamp int    `json:"lastCreatedAtTimestamp"`
		LastProcessedPoolId    string `json:"lastProcessedPoolID"`
	}
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
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
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := u.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, u.config.NewPoolLimit)
	if err != nil {
		return nil, metadataBytes, err
	}

	// Currently disable filter which will lead to dup process getnewpools, but it's oke
	// If we enable, we have to change logic of for loop in pool-service where "len(poolsList) < newPoolLimit"

	// subgraphPools = lo.Filter(subgraphPools, func(p SubgraphPool, _ int) bool {
	//	return p.ID != metadata.LastProcessedPoolId
	// })

	pools := make([]entity.Pool, 0, len(subgraphPools))

	chainID := valueobject.ChainID(u.config.ChainID)
	for _, p := range subgraphPools {
		token0Decimals, err := strconv.Atoi(p.Token0.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		token1Decimals, err := strconv.Atoi(p.Token1.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		tokens := []*entity.PoolToken{
			{
				Address:   p.Token0.ID,
				Swappable: true,
				Decimals:  uint8(token0Decimals),
				Name:      p.Token0.Name,
			},
			{
				Address:   p.Token1.ID,
				Swappable: true,
				Decimals:  uint8(token1Decimals),
				Name:      p.Token0.Name,
			},
		}
		for idx, token := range tokens {
			if token.Address == EMPTY_ADDRESS {
				tokens[idx].Address = strings.ToLower(valueobject.WrappedNativeMap[chainID])
			}
		}

		tickSpacing, err := strconv.Atoi(p.TickSpacing)
		if err != nil {
			return nil, metadataBytes, err
		}

		staticExtra := StaticExtra{
			PoolId:      p.ID,
			Currency0:   p.Token0.ID,
			Currency1:   p.Token1.ID,
			Fee:         p.Fee,
			TickSpacing: tickSpacing,

			HooksAddress:           common.HexToAddress(p.Hooks),
			UniversalRouterAddress: common.HexToAddress(u.config.UniversalRouterAddress),
			Permit2Address:         common.HexToAddress(u.config.Permit2Address),
			Multicall3Address:      common.HexToAddress(u.config.Multicall3Address),
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, metadataBytes, err
		}

		pool := entity.Pool{
			Address:     p.ID,
			Tokens:      tokens,
			Reserves:    entity.PoolReserves{"0", "0"},
			Exchange:    u.config.DexID,
			Type:        DexType,
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
			Timestamp:   time.Now().Unix(),
		}
		pools = append(pools, pool)
	}

	// Update metadata
	if len(subgraphPools) > 0 {
		lastCreatedAtTimestamp, err := strconv.Atoi(subgraphPools[len(subgraphPools)-1].CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, err
		}

		metadata.LastCreatedAtTimestamp = lastCreatedAtTimestamp
		metadata.LastProcessedPoolId = subgraphPools[len(subgraphPools)-1].ID
		metadataBytes, err = json.Marshal(metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	logger.WithFields(logger.Fields{
		"dexId": u.config.DexID,
		"pools": len(pools),
	}).Info("finished getting new pools")

	return pools, metadataBytes, nil
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp int, first int) ([]SubgraphPool, error) {
	req := graphqlpkg.NewRequest(getPoolsListQuery(lastCreatedAtTimestamp, first))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": d.config.DexID,
			"error": err,
		}).Errorf("failed to query subgraph")
		return nil, err
	}

	return response.Pools, nil
}
