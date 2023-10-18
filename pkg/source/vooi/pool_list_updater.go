package vooi

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

var (
	ErrFailedToGetLastIndex = errors.New("failed to get last index")
	ErrFailedToGetPoolState = errors.New("failed to get pool state")
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolListUpdaterMetadata struct {
		HasInitialized bool `json:"hasInitialized"`
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	pools := make([]entity.Pool, 0, len(u.config.PoolAddresses))

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      u.config.DexID,
					"duration_ms": time.Since(startTime).Milliseconds(),
					"pools":       lo.Map(pools, func(item entity.Pool, index int) string { return item.Address }),
				}).
			Debug("Finish getting new pools")
	}()

	var metadata PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}

		if metadata.HasInitialized {
			return nil, metadataBytes, nil
		}
	}

	for _, poolAddress := range u.config.PoolAddresses {
		pool, err := u.initPool(ctx, poolAddress)
		if err != nil {
			logger.
				WithFields(
					logger.Fields{
						"liquiditySource": DexTypeVooi,
						"poolAddress":     poolAddress,
						"error":           err,
					}).
				Warn("init pool failed")
			continue
		}

		pools = append(pools, pool)
	}

	newMetadataBytes, err := json.Marshal(PoolListUpdaterMetadata{HasInitialized: true})
	if err != nil {
		newMetadataBytes = []byte(`{"hasInitialized": true}`)
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPool(ctx context.Context, address string) (entity.Pool, error) {
	ctx = util.NewContextWithTimestamp(ctx)

	// Get lastIndex
	lastIndex, err := u.getLastIndex(ctx, address)
	if err != nil {
		return entity.Pool{}, err
	}

	var (
		paused bool
		a      *big.Int
		lpFee  *big.Int
		assets = make([]Asset, lastIndex)
	)

	getPoolState := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < lastIndex; i++ {
		getPoolState.AddCall(
			&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodIndexToAsset,
				Params: []interface{}{big.NewInt(int64(i))},
			}, []interface{}{&assets[i]})
	}

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: address,
		Method: poolMethodPaused,
		Params: nil,
	}, []interface{}{&paused})

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: address,
		Method: poolMethodLpFee,
		Params: nil,
	}, []interface{}{&lpFee})

	if _, err = getPoolState.TryAggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": DexTypeVooi,
					"poolAddress":     address,
					"error":           err,
				}).
			Error("failed to get pool state")

		return entity.Pool{}, ErrFailedToGetPoolState
	}

	poolTokens := make([]*entity.PoolToken, 0, len(assets))
	reserves := make([]string, 0, len(assets))
	assetByToken := make(map[string]Asset, len(assets))
	indexByToken := make(map[string]int, len(assets))

	for i, asset := range assets {
		token := strings.ToLower(asset.Token.Hex())

		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Decimals:  asset.Decimals,
			Swappable: true,
		})
		reserves = append(reserves, asset.Cash.String())
		assetByToken[token] = asset
		indexByToken[token] = i
	}

	poolExtra := PoolExtra{
		AssetByToken: assetByToken,
		IndexByToken: indexByToken,
		Paused:       paused,
		A:            a,
		LPFee:        lpFee,
	}

	poolExtraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:   strings.ToLower(address),
		Tokens:    poolTokens,
		Reserves:  reserves,
		Exchange:  u.config.DexID,
		Type:      DexTypeVooi,
		Extra:     string(poolExtraBytes),
		Timestamp: time.Now().Unix(),
	}, nil
}

func (u *PoolsListUpdater) getLastIndex(_ context.Context, address string) (int, error) {
	var lastIndex *big.Int

	getLastIndexRequest := u.ethrpcClient.NewRequest()
	getLastIndexRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: address,
		Method: poolMethodLastIndex,
		Params: nil,
	}, []interface{}{&lastIndex})

	if _, err := getLastIndexRequest.Call(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": DexTypeVooi,
					"poolAddress":     address,
					"error":           err,
				}).
			Error("failed to get lastIndex")

		return 0, ErrFailedToGetLastIndex
	}

	return int(lastIndex.Int64()), nil
}
