package nadfun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/goccy/go-json"
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

type Metadata struct {
	LastBlockTimestamp string `json:"lastBlockTimestamp"`
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) getCurvesList(ctx context.Context, lastBlockTimestamp *big.Int, first int) ([]SubgraphCurve, error) {
	req := graphqlpkg.NewRequest(getCurvesQuery(lastBlockTimestamp, first))

	var response struct {
		Curves []SubgraphCurve `json:"curves"`
	}

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Curves, nil
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastBlockTimestamp: "0",
	}
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	lastBlockTimestamp, ok := new(big.Int).SetString(metadata.LastBlockTimestamp, 10)
	if !ok {
		lastBlockTimestamp = integer.Zero()
	}

	batchSize := u.config.NewPoolLimit
	if batchSize <= 0 || batchSize > graphFirstLimit {
		batchSize = graphFirstLimit
	}

	curves, err := u.getCurvesList(ctx, lastBlockTimestamp, batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(curves))
	for _, curve := range curves {
		pool, err := u.createPool(curve)
		if err != nil {
			continue
		}

		pools = append(pools, pool)
	}

	newLastBlockTimestamp := metadata.LastBlockTimestamp
	if len(curves) > 0 {
		lastCurveIndex := len(curves) - 1
		newLastBlockTimestamp = curves[lastCurveIndex].BlockTimestamp
	}

	newMetadata := Metadata{
		LastBlockTimestamp: newLastBlockTimestamp,
	}
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) createPool(curve SubgraphCurve) (entity.Pool, error) {
	wNative := valueobject.WrappedNativeMap[u.config.ChainID]

	staticExtra := StaticExtra{
		Router: u.config.RouterAddress,
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:   GetPoolAddress(curve.Token),
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   wNative,
				Swappable: true,
			},
			{
				Address:   curve.Token,
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
	}, nil
}
