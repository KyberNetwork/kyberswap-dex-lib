package poolmanager

import (
	"context"

	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type IPoolFactory interface {
	NewPoolByAddress(ctx context.Context, pools []*entity.Pool) map[string]poolpkg.IPool
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
}
