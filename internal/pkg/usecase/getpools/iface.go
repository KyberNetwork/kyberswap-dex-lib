package getpools

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

//go:generate mockgen -destination ../../mocks/usecase/getpools/pool_repository.go -package poolmanager github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools IPoolRepository
type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Pool, error)
	FindAllAddresses(ctx context.Context) ([]string, error)
}
