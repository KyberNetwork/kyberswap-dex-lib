package skypsm

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolListUpdater struct {
	ethrpcClient   *ethrpc.Client
	config         *Config
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:         config,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (u *PoolListUpdater) GetNewPools(_ context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized || metadataBytes != nil {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
	}).Info("Started getting new pool")

	var rateProvider, pocket common.Address
	if _, err := u.ethrpcClient.NewRequest().
		SetContext(context.Background()).AddCall(&ethrpc.Call{
		ABI:    psm3ABI,
		Target: u.config.PsmAddress,
		Method: psm3MethodRateProvider,
	}, []interface{}{&rateProvider}).AddCall(&ethrpc.Call{
		ABI:    psm3ABI,
		Target: u.config.PsmAddress,
		Method: psm3MethodPocket,
	}, []interface{}{&pocket}).Aggregate(); err != nil {
		return nil, nil, err
	}

	tokens := make(entity.PoolTokens, len(u.config.Tokens))
	reserves := make(entity.PoolReserves, len(u.config.Tokens))
	for i, token := range u.config.Tokens {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(token),
			Swappable: true,
		}
		reserves[i] = defaultReserves
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		RateProvider: rateProvider.Hex(),
		Pocket:       pocket,
	})
	if err != nil {
		return nil, nil, err
	}

	poolEntity := entity.Pool{
		Address:     strings.ToLower(u.config.PsmAddress),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		StaticExtra: string(staticExtraBytes),
	}

	u.hasInitialized = true

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
		"address":  poolEntity.Address,
	}).Info("Finished getting new pool")

	return []entity.Pool{poolEntity}, nil, nil
}
