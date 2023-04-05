package job

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

// IGetAllPoolAddressesUseCase get all pool addresses from Redis
type IGetAllPoolAddressesUseCase interface {
	Handle(ctx context.Context) ([]string, error)
}

// IIndexPoolsUseCase get pools info save/update into Redis sorted set, score by reserveUsd or amplifiedTvl
type IIndexPoolsUseCase interface {
	Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult
}
