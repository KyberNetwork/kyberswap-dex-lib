package skypsm

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

func (u *PoolListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
	}).Info("Started getting new pool")

	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		return nil, nil, errors.New("misconfigured poolPath")
	}
	var initialPool InitialPool
	if err := json.Unmarshal(byteData, &initialPool); err != nil {
		return nil, nil, err
	}

	tokens := make(entity.PoolTokens, 0, len(initialPool.Tokens))
	reserves := make(entity.PoolReserves, 0, len(initialPool.Tokens))
	for _, token := range initialPool.Tokens {
		tokenEntity := entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Name:      token.Name,
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: true,
		}
		tokens = append(tokens, &tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	staticExtra := StaticExtra{
		RateProvider: initialPool.RateProvider,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, nil, err
	}

	poolEntity := entity.Pool{
		Address:     strings.ToLower(initialPool.ID),
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
