package v3

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type (
	PoolsListUpdater struct {
		config            *Config
		ethrpcClient      *ethrpc.Client
		graphqlClient     *graphqlpkg.Client
		staticPoolsLoaded bool
	}

	PoolsListUpdaterMetadata struct {
		LastBlockNumber uint64 `json:"lastBlockNumber"`
	}

	SubgraphToken struct {
		Address string `json:"address"`
	}

	SubgraphMarket struct {
		ID string `json:"id"`
	}

	SubgraphBaseToken struct {
		Market              SubgraphMarket `json:"market"`
		Token               SubgraphToken  `json:"token"`
		CreationBlockNumber string         `json:"creationBlockNumber"`
	}

	SubgraphResponse struct {
		BaseTokens []SubgraphBaseToken `json:"baseTokens"`
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

	pools, newMetadataBytes, err := u.collectNewPools(ctx, metadataBytes)
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"new_pools":   len(pools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) collectNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	// Use static config if available, otherwise use subgraph
	if len(u.config.Pairs) > 0 {
		return u.loadStaticPools()
	}

	return u.loadSubgraphPools(ctx, metadataBytes)
}

func (u *PoolsListUpdater) loadStaticPools() ([]entity.Pool, []byte, error) {
	if u.staticPoolsLoaded {
		return []entity.Pool{}, nil, nil
	}

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID,
		"count":  len(u.config.Pairs),
	}).Info("Loading static pools")

	staticPools := u.createPoolsFromConfig()
	u.staticPoolsLoaded = true

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID,
		"loaded": len(staticPools),
	}).Info("Static pools loaded successfully")

	return staticPools, nil, nil
}

func (u *PoolsListUpdater) loadSubgraphPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.graphqlClient == nil {
		logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Error("GraphQL client is not set")
		return []entity.Pool{}, metadataBytes, nil
	}

	lastBlockNumber, err := u.getLastBlockNumber(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.config.DexID, "err": err}).
			Warn("getLastBlockNumber failed, using 0")
		lastBlockNumber = 0
	}

	subgraphData, err := u.fetchSubgraphData(ctx, lastBlockNumber)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.config.DexID, "err": err}).
			Error("fetchSubgraphData failed")
		return nil, metadataBytes, err
	}

	if len(subgraphData.BaseTokens) == 0 {
		return []entity.Pool{}, metadataBytes, nil
	}

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID,
		"count":  len(subgraphData.BaseTokens),
	}).Info("Processing subgraph data")

	subgraphPools, newLastBlockNumber, err := u.createPoolsFromSubgraph(subgraphData)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.config.DexID, "err": err}).
			Error("createPoolsFromSubgraph failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(newLastBlockNumber)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.config.DexID, "err": err}).
			Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID,
		"loaded": len(subgraphPools),
	}).Info("Subgraph pools loaded successfully")

	return subgraphPools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) fetchSubgraphData(ctx context.Context, lastBlockNumber uint64) (*SubgraphResponse, error) {
	req := graphqlpkg.NewRequest(fmt.Sprintf(`{
	baseTokens(
		where: { creationBlockNumber_gt: %d }
	) {
		market {
			id
		}
		token {
			address
		}
		creationBlockNumber
	}
}`, lastBlockNumber))

	var resp SubgraphResponse
	if err := u.graphqlClient.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (u *PoolsListUpdater) getLastBlockNumber(metadataBytes []byte) (uint64, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.LastBlockNumber, nil
}

func (u *PoolsListUpdater) createPoolsFromConfig() []entity.Pool {
	pools := make([]entity.Pool, 0, len(u.config.Pairs))

	for marketID, baseToken := range u.config.Pairs {
		pool, err := u.createPool(marketID, baseToken)
		if err != nil {
			logger.WithFields(logger.Fields{"market_id": marketID, "err": err}).Error("createPool failed")
			continue
		}

		pools = append(pools, pool)
	}

	return pools
}

func (u *PoolsListUpdater) createPoolsFromSubgraph(data *SubgraphResponse) ([]entity.Pool, uint64, error) {
	pools := make([]entity.Pool, 0, len(data.BaseTokens))
	var newLastBlockNumber uint64

	for _, baseToken := range data.BaseTokens {
		pool, err := u.createPool(baseToken.Market.ID, baseToken.Token.Address)
		if err != nil {
			logger.WithFields(logger.Fields{"market_id": baseToken.Market.ID, "err": err}).Error("createPool failed")
			continue
		}

		pools = append(pools, pool)

		if blockNum := parseUint64(baseToken.CreationBlockNumber); blockNum > newLastBlockNumber {
			newLastBlockNumber = blockNum
		}
	}

	return pools, newLastBlockNumber, nil
}

func (u *PoolsListUpdater) createPool(cToken, baseToken string) (entity.Pool, error) {
	cTokenAddr := strings.ToLower(cToken)
	baseTokenAddr := strings.ToLower(baseToken)

	allTokens := []*entity.PoolToken{
		{
			Address:   cTokenAddr,
			Swappable: true,
		},
		{
			Address:   baseTokenAddr,
			Swappable: true,
		},
	}

	newPool := entity.Pool{
		Address:   cTokenAddr,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens:    allTokens,
	}

	return newPool, nil
}

func (u *PoolsListUpdater) newMetadata(newLastBlockNumber uint64) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		LastBlockNumber: newLastBlockNumber,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
