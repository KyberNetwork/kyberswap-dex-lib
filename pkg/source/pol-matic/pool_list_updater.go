package polmatic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/account"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

var (
	ErrFailedToGetTokens        = errors.New("failed to get tokens")
	ErrTokenIsNotSet            = errors.New("token is not set")
	ErrFailedToGetTokenDecimals = errors.New("failed to get token decimals")
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
	ctx = util.NewContextWithTimestamp(ctx)
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      u.config.DexID,
					"duration_ms": time.Since(startTime).Milliseconds(),
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

	var (
		matic   common.Address
		polygon common.Address
	)

	getTokens := u.ethrpcClient.NewRequest().SetContext(ctx)
	getTokens.AddCall(
		&ethrpc.Call{
			ABI:    polygonMigrationABI,
			Target: u.config.PolygonMigrationAddress,
			Method: polygonMigrationMethodMatic,
			Params: []interface{}{},
		}, []interface{}{&matic})
	getTokens.AddCall(
		&ethrpc.Call{
			ABI:    polygonMigrationABI,
			Target: u.config.PolygonMigrationAddress,
			Method: polygonMigrationMethodPolygon,
			Params: []interface{}{},
		}, []interface{}{&polygon})

	if _, err := getTokens.TryAggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": u.config.DexID,
					"poolAddress":     u.config.PolygonMigrationAddress,
					"error":           err,
				}).
			Error("failed to get tokens")

		return nil, nil, ErrFailedToGetTokens
	}

	if account.IsZeroAddress(matic) || account.IsZeroAddress(polygon) {
		return nil, nil, ErrTokenIsNotSet
	}

	var (
		polygonDecimals uint8
		maticDecimals   uint8
	)

	getTokenDecimals := u.ethrpcClient.NewRequest().SetContext(ctx)
	getTokenDecimals.AddCall(
		&ethrpc.Call{
			ABI:    erc20ABI,
			Target: matic.String(),
			Method: erc20MethodDecimals,
			Params: []interface{}{},
		}, []interface{}{&maticDecimals})
	getTokenDecimals.AddCall(
		&ethrpc.Call{
			ABI:    erc20ABI,
			Target: polygon.String(),
			Method: erc20MethodDecimals,
			Params: []interface{}{},
		}, []interface{}{&polygonDecimals})
	if _, err := getTokenDecimals.TryAggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": u.config.DexID,
					"poolAddress":     u.config.PolygonMigrationAddress,
					"error":           err,
				}).
			Error("failed to get token decimals")

		return nil, nil, ErrFailedToGetTokenDecimals
	}

	newMetadataBytes, err := json.Marshal(PoolListUpdaterMetadata{HasInitialized: true})
	if err != nil {
		newMetadataBytes = []byte(`{"hasInitialized": true}`)
	}

	// Token0 has to be Matic, otherwise it will break swap logic
	return []entity.Pool{
		{
			Address: strings.ToLower(u.config.PolygonMigrationAddress),
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(matic.String()), Decimals: maticDecimals, Swappable: true},
				{Address: strings.ToLower(polygon.String()), Decimals: polygonDecimals, Swappable: true},
			},
			Reserves:  []string{"0", "0"},
			Exchange:  u.config.DexID,
			Type:      DexTypePolMatic,
			Timestamp: time.Now().Unix(),
		},
	}, newMetadataBytes, nil
}
