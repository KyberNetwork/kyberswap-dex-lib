package synthereum

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

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, nil, errors.New("misconfigured poolPath")
	}

	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(poolItems))
	for i := range poolItems {
		poolEntity, err := u.getNewPool(&poolItems[i])
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, poolEntity)
	}

	u.hasInitialized = true
	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
		"pools":    len(pools),
	}).Info("finish fetching pools")

	return pools, nil, nil
}

func (u *PoolsListUpdater) getNewPool(item *PoolItem) (entity.Pool, error) {
	tokens := make([]*entity.PoolToken, 0, len(item.Tokens))
	reserves := make(entity.PoolReserves, 0, len(item.Tokens))
	for _, token := range item.Tokens {
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: true,
		})
		reserves = append(reserves, reserveZero)
	}

	staticExtra := StaticExtra{
		PoolType: item.PoolType,
		Vault:    strings.ToLower(item.Vault),
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     strings.ToLower(item.ID),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}, nil
}
