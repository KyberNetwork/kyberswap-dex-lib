package v2

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
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	PoolsListUpdaterMetadata struct {
		LastBlockNumber uint64 `json:"lastBlockNumber"`
	}

	SubgraphMarket struct {
		ID                  string `json:"id"`
		UnderlyingAddress   string `json:"underlyingAddress"`
		CreationBlockNumber string `json:"creationBlockNumber"`
	}

	SubgraphResponse struct {
		Markets []SubgraphMarket `json:"markets"`
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

	lastBlockNumber, err := u.getLastBlockNumber(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getLastBlockNumber failed")
	}

	subgraphData, err := u.fetchAllData(ctx, lastBlockNumber)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("fetchAllData failed")

		return nil, metadataBytes, err
	}

	if len(subgraphData.Markets) == 0 {
		return []entity.Pool{}, metadataBytes, nil
	}

	pools, newLastBlockNumber, err := u.createPoolsFromData(subgraphData)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("createPoolsFromDataAndFindLastBlock failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(newLastBlockNumber)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

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

func (u *PoolsListUpdater) fetchAllData(ctx context.Context, lastBlockNumber uint64) (*SubgraphResponse, error) {
	req := graphqlpkg.NewRequest(fmt.Sprintf(`{
	markets(
		where: { creationBlockNumber_gt: %d }
	) {
		id
		underlyingAddress
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

func (u *PoolsListUpdater) createPoolsFromData(data *SubgraphResponse) ([]entity.Pool, uint64, error) {
	pools := make([]entity.Pool, 0)
	var newLastBlockNumber uint64

	for _, market := range data.Markets {
		pool, err := u.createPool(market.ID, market.UnderlyingAddress)
		if err != nil {
			logger.WithFields(logger.Fields{"market_id": market.ID, "err": err}).Error("createPool failed")
			continue
		}

		pools = append(pools, pool)

		if blockNum := parseUint64(market.CreationBlockNumber); blockNum > newLastBlockNumber {
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

	reserves := []string{"0", "0"}

	newPool := entity.Pool{
		Address:   cTokenAddr,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
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
