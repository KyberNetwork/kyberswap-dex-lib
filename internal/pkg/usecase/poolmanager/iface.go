package poolmanager

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPoolSimulator
	NewPools(ctx context.Context, pools []*entity.Pool) []poolpkg.IPoolSimulator
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}

type IPoolRankRepository interface {
	FindGlobalBestPools(ctx context.Context, poolCount int64) []string
}
