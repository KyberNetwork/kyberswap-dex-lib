package poolmanager

import (
	"context"

	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPool
	NewPools(ctx context.Context, pools []*entity.Pool) []poolpkg.IPool
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

type IPoolRankRepository interface {
	FindGlobalBestPools(ctx context.Context, poolCount int64) []string
}
