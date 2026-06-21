package machima

import (
	"context"
	"math/big"
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
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

type Metadata struct {
	LastCreatedAt *big.Int `json:"lastCreatedAt"`
}

// SubgraphPool matches elixir-subgraph schema: Pool entity
type SubgraphPool struct {
	ID           string        `json:"id"`
	Token0       SubgraphToken `json:"token0"`
	Token1       SubgraphToken `json:"token1"`
	CounterAsset string        `json:"counterAsset"`
	TradedToken  string        `json:"tradedToken"`
	CreatedAt    string        `json:"createdAt"`
}

type SubgraphToken struct {
	Address  string `json:"id"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

var _ = poollist.RegisterFactoryCEG(DexTypeMachima, NewPoolsListUpdater)

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

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAt: big.NewInt(0),
	}
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAt)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get machima pools list from subgraph")
		return nil, metadataBytes, err
	}

	logger.Infof("got %v machima subgraph pools", len(subgraphPools))

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		token0 := strings.ToLower(p.Token0.Address)
		token1 := strings.ToLower(p.Token1.Address)
		counterAsset := strings.ToLower(p.CounterAsset)
		token := strings.ToLower(p.TradedToken)

		// Validate counter asset
		if counterAsset == "" || token == "" {
			continue
		}

		staticExtra, _ := json.Marshal(StaticExtra{
			CounterAsset:  counterAsset,
			Token:         token,
			RouterAddress: d.config.RouterAddress,
			WETH:          strings.ToLower(d.config.WETH),
			USDC:          strings.ToLower(d.config.USDC),
			XMA:           strings.ToLower(d.config.XMA),
		})

		// Subgraph reserves are BigDecimal (not wei); leave as "0" and let the
		// tracker populate real wei values from balanceOf on first cycle.
		reserves := []string{"0", "0"}

		pools = append(pools, entity.Pool{
			Address:   p.ID,
			SwapFee:   float64(PoolFee),
			Exchange:  DexTypeMachima,
			Type:      DexTypeMachima,
			Timestamp: parseTimestamp(p.CreatedAt),
			Reserves:  reserves,
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			StaticExtra: string(staticExtra),
		})
	}

	var lastCreatedAt = metadata.LastCreatedAt
	if len(subgraphPools) > 0 {
		ts, ok := new(big.Int).SetString(subgraphPools[len(subgraphPools)-1].CreatedAt, 10)
		if ok {
			lastCreatedAt = ts
		}
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAt: lastCreatedAt,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAt *big.Int) ([]SubgraphPool, error) {
	query := `{
		pools(
			first: 1000,
			orderBy: createdAt,
			orderDirection: asc,
			where: { createdAt_gt: "` + lastCreatedAt.String() + `" }
		) {
			id
			token0 { id symbol decimals }
			token1 { id symbol decimals }
			counterAsset
			tradedToken
			createdAt
		}
	}`

	req := graphqlpkg.NewRequest(query)

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Pools, nil
}

func parseTimestamp(s string) int64 {
	ts, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return 0
	}
	return ts.Int64()
}
