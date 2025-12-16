package gateway

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// Register this liquidity source factory
var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("[%s] discovering pools", DexType)

	// Define the tokens in this "pool"
	// All individual operations supported: USDC ↔ iUSD ↔ siUSD ↔ liUSD (except liUSD -> iUSD given async redemption)
	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(u.config.USDC),
			Swappable: true,
		},
		{
			Address:   strings.ToLower(u.config.IUSD),
			Swappable: true,
		},
		{
			Address:   strings.ToLower(u.config.SIUSD),
			Swappable: true,
		},
	}

	// Add all liUSD tokens (one per unwinding epoch)
	for _, liusd := range u.config.LIUSDTokens {
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(liusd.Address),
			Swappable: true,
		})
	}

	// Initialize reserves (will be updated by pool tracker)
	reserves := make([]string, len(tokens))
	for i := range reserves {
		reserves[i] = "0"
	}

	pool := entity.Pool{
		Address:   strings.ToLower(u.config.Gateway),
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
	}

	// Get initial pool state
	poolWithState, err := getPoolState(ctx, u.ethrpcClient, u.config, &pool)
	if err != nil {
		logger.WithFields(logger.Fields{
			"gateway": u.config.Gateway,
			"error":   err,
		}).Errorf("failed to get initial pool state")
		return nil, nil, err
	}

	return []entity.Pool{poolWithState}, nil, nil
}
