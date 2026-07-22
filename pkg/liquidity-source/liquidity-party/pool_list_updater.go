package liquidityparty

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		// Offset is the number of pools already discovered from the PartyPlanner index.
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools is the cold-start backfill for pool discovery. The primary path is the PartyStarted
// event decoded in pool_factory.go, which surfaces a pool the block it is created; this paging over
// poolCount()/getAllPools(offset,limit) with a monotonic offset cursor only backfills pools created
// before the event subscription began. Pools are admin-created and never leave the index, so a
// monotonic cursor is sound; once caught up this returns nothing and the event path carries new
// pools. Killed pools are still returned here and are disabled later by the tracker (killing is
// irreversible).
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	dexID := u.config.DexID
	startTime := time.Now()
	lg := logger.WithFields(logger.Fields{"dex_id": dexID})
	lg.Info("Started getting new pools")

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		lg.WithFields(logger.Fields{"err": err}).Warn("getMetadata failed")
	}

	poolCount, err := u.getPoolCount(ctx)
	if err != nil {
		lg.WithFields(logger.Fields{"err": err}).Error("poolCount failed")
		return nil, metadataBytes, err
	}

	if metadata.Offset >= poolCount {
		return nil, metadataBytes, nil
	}

	batchSize := min(poolListBatchSize, poolCount-metadata.Offset)

	poolAddresses, err := u.listPoolAddresses(ctx, metadata.Offset, batchSize)
	if err != nil {
		lg.WithFields(logger.Fields{"err": err}).Error("listPoolAddresses failed")
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, poolAddresses)
	if err != nil {
		lg.WithFields(logger.Fields{"err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: metadata.Offset + len(poolAddresses)})
	if err != nil {
		return nil, metadataBytes, err
	}

	lg.WithFields(logger.Fields{
		"pools_len":   len(pools),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getPoolCount(ctx context.Context) (int, error) {
	var poolCount *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    partyPlannerABI,
		Target: u.config.PartyPlannerAddress,
		Method: plannerMethodPoolCount,
	}, []any{&poolCount})

	if _, err := req.Call(); err != nil {
		return 0, err
	}
	return int(poolCount.Int64()), nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset, limit int) ([]common.Address, error) {
	var pools []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    partyPlannerABI,
		Target: u.config.PartyPlannerAddress,
		Method: plannerMethodGetAllPools,
		Params: []any{big.NewInt(int64(offset)), big.NewInt(int64(limit))},
	}, []any{&pools})

	if _, err := req.Call(); err != nil {
		return nil, err
	}
	return pools, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	tokensByPool, err := u.listPoolTokens(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))
	now := time.Now().Unix()
	for i, poolAddress := range poolAddresses {
		tokens := make([]*entity.PoolToken, 0, len(tokensByPool[i]))
		reserves := make(entity.PoolReserves, 0, len(tokensByPool[i]))
		for _, token := range tokensByPool[i] {
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(token.Hex()),
				Swappable: true,
			})
			reserves = append(reserves, "0")
		}

		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(poolAddress.Hex()),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  reserves,
			Tokens:    tokens,
		})
	}

	return pools, nil
}

func (u *PoolsListUpdater) listPoolTokens(ctx context.Context, poolAddresses []common.Address) ([][]common.Address, error) {
	poolTokens := make([][]common.Address, len(poolAddresses))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    partyPoolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodAllTokens,
		}, []any{&poolTokens[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return poolTokens, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (PoolsListUpdaterMetadata, error) {
	if len(metadataBytes) == 0 {
		return PoolsListUpdaterMetadata{}, nil
	}
	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}
	return metadata, nil
}
