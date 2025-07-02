package poolparty

import (
	"context"
	"fmt"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphql.Client
}

var _ = poollist.RegisterFactoryCG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	graphqlClient *graphql.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAtTimestamp: 0,
	}
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, metadata.LastPoolIds)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pools list from subgraph")
		return nil, metadataBytes, err
	}

	numSubgraphPools := len(subgraphPools)

	if numSubgraphPools == 0 {
		return nil, metadataBytes, nil
	}

	// Track the last pool's CreatedAtTimestamp
	var lastPoolIds []string
	lastCreatedAtTimestampStr := subgraphPools[numSubgraphPools-1].CreatedAtTimestamp
	lastCreatedAtTimestamp, err := kutils.Atoi[int](lastCreatedAtTimestampStr)
	if err != nil {
		return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v, error: %v",
			lastCreatedAtTimestampStr, subgraphPools[numSubgraphPools-1].ID, err)
	}

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		tokenNative := entity.PoolToken{
			Address:   strings.ToLower(valueobject.WrappedNativeMap[d.config.ChainID]),
			Symbol:    "WETH",
			Decimals:  18,
			Swappable: true,
		}

		tokenDecimals, _ := kutils.Atoi[int](p.TokenDecimals)
		tokenTarget := entity.PoolToken{
			Address:   strings.ToLower(p.TokenAddress),
			Symbol:    p.TokenSymbol,
			Decimals:  uint8(tokenDecimals),
			Swappable: true,
		}

		tokens := []*entity.PoolToken{&tokenNative, &tokenTarget}
		reserves := []string{"0", p.PublicAmountAvailable}

		createdAtTimestamp, err := kutils.Atoi[int64](p.CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v", p.CreatedAtTimestamp, p.ID)
		}

		extra := Extra{
			PoolStatus:            p.PoolStatus,
			IsVisible:             p.IsVisible,
			BoostPriceBps:         d.config.BoostPriceBps,
			PublicAmountAvailable: bignumber.NewBig10(p.PublicAmountAvailable),
		}
		extraBytes, _ := json.Marshal(extra)

		var newPool = entity.Pool{
			Address:   p.ID,
			Exchange:  d.config.DexID,
			Type:      DexType,
			Timestamp: createdAtTimestamp,
			Reserves:  reserves,
			Tokens:    tokens,
			Extra:     string(extraBytes),
		}

		pools = append(pools, newPool)
		if p.CreatedAtTimestamp == lastCreatedAtTimestampStr {
			lastPoolIds = append(lastPoolIds, p.ID)
		}
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
		LastPoolIds:            lastPoolIds,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("got %v %v pools", len(pools), d.config.DexID)

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getPoolsList(
	ctx context.Context,
	lastCreatedAtTimestamp int,
	lastPoolIds []string,
) ([]SubgraphPool, error) {
	req := graphql.NewRequest(getPoolsListQuery(lastCreatedAtTimestamp, lastPoolIds))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"dex":   d.config.DexID,
			"error": err,
		}).Errorf("failed to query subgraph")
		return nil, err
	}

	return response.Pools, nil
}
