package mkr_sky

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater handles updating the list of pools for MKR-SKY DEX
type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

// NewPoolsListUpdater creates a new PoolsListUpdater instance
func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools retrieves new pools and updates the pool list
func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	pools, err := d.initPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to initialize pools")
		return nil, nil, err
	}

	logger.WithFields(logger.Fields{
		"poolCount": len(pools),
	}).Info("successfully fetched pools")

	return pools, nil, nil
}

// initPools initializes the pools from the configured pool path
func (d *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	byteData, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		return nil, fmt.Errorf("misconfigured poolPath: pool data not found for path %s", d.config.PoolPath)
	}

	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool data: %w", err)
	}

	if len(poolItems) == 0 {
		return nil, errors.New("no pools found in configuration")
	}

	// Validate all pool items before processing
	for i, item := range poolItems {
		if err := item.Validate(); err != nil {
			return nil, fmt.Errorf("invalid pool item at index %d: %w", i, err)
		}
	}

	pools, err := d.processBatch(ctx, poolItems)
	if err != nil {
		return nil, fmt.Errorf("failed to process pool batch: %w", err)
	}

	d.hasInitialized = true
	return pools, nil
}

// processBatch processes a batch of pool items
func (d *PoolsListUpdater) processBatch(ctx context.Context, poolItems []PoolItem) ([]entity.Pool, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	pools := make([]entity.Pool, 0, len(poolItems))
	errors := make([]error, 0, len(poolItems))

	for _, pool := range poolItems {
		poolEntity, err := d.getNewPool(ctx, &pool)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to get pool %s: %w", pool.ID, err))
			continue
		}
		pools = append(pools, poolEntity)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d errors while processing pools: %v", len(errors), errors)
	}

	return pools, nil
}

// getNewPool creates a new pool entity from a pool item
func (d *PoolsListUpdater) getNewPool(ctx context.Context, pool *PoolItem) (entity.Pool, error) {
	if ctx.Err() != nil {
		return entity.Pool{}, fmt.Errorf("context error: %w", ctx.Err())
	}

	if pool == nil {
		return entity.Pool{}, errors.New("pool item is nil")
	}

	tokens, reserves := preparePoolTokens(pool.Tokens)
	rate, err := d.fetchPoolRate(ctx, pool.ID)
	if err != nil {
		return entity.Pool{}, fmt.Errorf("failed to fetch pool rate: %w", err)
	}

	staticExtra := StaticExtra{Rate: rate}
	if err := staticExtra.Validate(); err != nil {
		return entity.Pool{}, fmt.Errorf("invalid static extra: %w", err)
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, fmt.Errorf("failed to marshal static extra: %w", err)
	}

	return entity.Pool{
		Address:     pool.ID,
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}, nil
}

// preparePoolTokens prepares the tokens and reserves arrays for a pool
func preparePoolTokens(poolTokens []entity.PoolToken) ([]*entity.PoolToken, entity.PoolReserves) {
	tokens := make([]*entity.PoolToken, 0, len(poolTokens))
	reserves := make(entity.PoolReserves, 0, len(poolTokens))

	for _, token := range poolTokens {
		swappable := !strings.EqualFold(token.Address, SkyAddress)
		tokenEntity := &entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: swappable,
		}
		tokens = append(tokens, tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	return tokens, reserves
}

// fetchPoolRate fetches the rate for a pool
func (d *PoolsListUpdater) fetchPoolRate(ctx context.Context, poolID string) (*big.Int, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	}

	var rate *big.Int
	req := d.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    mkrSkyABI,
		Target: poolID,
		Method: "rate",
	}, []any{&rate})

	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to fetch rate: %w", err)
	}

	return rate, nil
}
