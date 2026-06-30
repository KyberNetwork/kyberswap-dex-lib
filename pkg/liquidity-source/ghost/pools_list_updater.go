package ghost

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

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.initPools()
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Errorf("[%s] failed to init pools", DexType)
		return nil, nil, err
	}

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools() ([]entity.Pool, error) {
	byteData, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		return nil, errors.New("misconfigured poolPath")
	}

	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolItems))
	for i := range poolItems {
		p, err := d.buildPool(&poolItems[i])
		if err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}

	d.hasInitialized = true

	return pools, nil
}

func (d *PoolsListUpdater) buildPool(item *PoolItem) (entity.Pool, error) {
	tokens := make([]*entity.PoolToken, 0, len(item.Tokens))
	reserves := make(entity.PoolReserves, 0, len(item.Tokens))

	for _, tok := range item.Tokens {
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(tok.Address),
			Swappable: true,
		})
		reserves = append(reserves, defaultReserves)
	}

	staticExtraBytes, err := json.Marshal(item.StaticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     item.ID,
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
	}, nil
}
