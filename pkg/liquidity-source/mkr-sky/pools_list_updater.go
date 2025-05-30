package mkr_sky

import (
	"context"
	"errors"
	"math/big"
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
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	pools, err := d.initPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	byteData, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, errors.New("misconfigured poolPath")
	}
	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, err
	}

	pools, err := d.processBatch(ctx, poolItems)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to processBatch")
		return nil, err
	}
	d.hasInitialized = true

	return pools, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolItems []PoolItem) ([]entity.Pool, error) {
	var pools = make([]entity.Pool, 0, len(poolItems))

	for _, pool := range poolItems {
		var err error
		var poolEntity entity.Pool

		poolEntity, err = d.getNewPool(ctx, &pool)

		if err != nil {
			return nil, err
		}

		pools = append(pools, poolEntity)
	}

	return pools, nil
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, pool *PoolItem) (entity.Pool, error) {
	var (
		tokens   = make([]*entity.PoolToken, 0, len(pool.Tokens))
		reserves = make(entity.PoolReserves, 0, len(pool.Tokens))
		rate     *big.Int
	)

	req := d.ethrpcClient.R().SetContext(ctx)
	for _, token := range pool.Tokens {
		tokenEntity := entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: true,
		}

		tokens = append(tokens, &tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    mkrSkyABI,
		Target: pool.ID,
		Method: "rate",
	}, []any{&rate})

	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Rate: rate,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	poolEntity := entity.Pool{
		Address:     pool.ID,
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}

	return poolEntity, nil
}
