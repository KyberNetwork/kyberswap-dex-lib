package getpools

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/pkg/mempool"
)

type GetPoolsUseCase struct {
	poolRepo IPoolRepository
}

func NewGetPoolsUseCase(
	poolRepo IPoolRepository,
) *GetPoolsUseCase {
	return &GetPoolsUseCase{
		poolRepo: poolRepo,
	}
}

func (u *GetPoolsUseCase) Handle(ctx context.Context, query dto.GetPoolsQuery) (*dto.GetPoolsResult, error) {
	pools, err := u.poolRepo.FindByAddresses(ctx, query.IDs)
	if err != nil {
		return nil, err
	}

	result := dto.NewGetPoolsResult(pools)
	defer mempool.ReserveMany(pools...)
	return result, nil
}
