package baseline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphqlpkg.Client
}

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	_ *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

type SubgraphBToken struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	Symbol     string `json:"symbol"`
	Decimals   int    `json:"decimals"`
	DeployedAt int64  `json:"deployedAt"`
	Reserve    struct {
		Address  string `json:"address"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
	} `json:"reserve"`
}

func (d *PoolsListUpdater) getBTokensList(ctx context.Context, offset, limit int) ([]SubgraphBToken, error) {
	if d.graphqlClient == nil {
		return nil, errors.New("graphql client is not configured")
	}

	req := graphqlpkg.NewRequest(fmt.Sprintf(`{
		bTokens(
			filter: { chainId: "%d" }
			sortBy: { field: DEPLOYED_AT, direction: ASC }
			limit: %d
			offset: %d
		) {
			items {
				address
				name
				symbol
				decimals
				deployedAt
				reserve {
					address
					name
					symbol
					decimals
				}
			}
		}
	}`, d.config.ChainID, limit, offset))

	var response struct {
		BTokens struct {
			Items []SubgraphBToken `json:"items"`
		} `json:"bTokens"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.BTokens.Items, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	limit := d.config.NewPoolLimit
	if limit <= 0 {
		limit = 20
	}

	bTokens, err := d.getBTokensList(ctx, metadata.Offset, limit)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("[Baseline] failed to query bTokens from API")
		return nil, metadataBytes, err
	}

	if len(bTokens) == 0 {
		return nil, metadataBytes, nil
	}

	pools := make([]entity.Pool, 0, len(bTokens))
	for _, bt := range bTokens {
		bTokenAddr := strings.ToLower(bt.Address)
		reserveAddr := strings.ToLower(bt.Reserve.Address)

		extra := Extra{
			RelayAddress: d.config.RelayAddress,
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return nil, metadataBytes, err
		}

		pools = append(pools, entity.Pool{
			Address:   bTokenAddr,
			Exchange:  d.config.DexID,
			Type:      DexType,
			Timestamp: bt.DeployedAt,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: reserveAddr, Decimals: uint8(bt.Reserve.Decimals), Symbol: bt.Reserve.Symbol, Swappable: true},
				{Address: bTokenAddr, Decimals: uint8(bt.Decimals), Symbol: bt.Symbol, Swappable: true},
			},
			Extra: string(extraBytes),
		})
	}

	newMetadataBytes, err := json.Marshal(Metadata{Offset: metadata.Offset + len(bTokens)})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("[Baseline] got %d pools from API", len(pools))

	return pools, newMetadataBytes, nil
}
