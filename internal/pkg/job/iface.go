package job

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/generatepath"
)

// IGetAllPoolAddressesUseCase get all pool addresses from Redis
type IGetAllPoolAddressesUseCase interface {
	Handle(ctx context.Context) ([]string, error)
}

// IIndexPoolsUseCase get pools info save/update into Redis sorted set, score by reserveUsd or amplifiedTvl
type IIndexPoolsUseCase interface {
	Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult
}

// IUpdateSuggestedGasPriceUseCase get suggested gas price from rpc and save it to Redis
type IUpdateSuggestedGasPriceUseCase interface {
	Handle(ctx context.Context) (*dto.UpdateGasPriceResult, error)
}

// IGeneratePathUseCase generates the best paths from configured token pairs & amount and save them to Redis
type IGeneratePathUseCase interface {
	Handle(ctx context.Context)
	ApplyConfig(config generatepath.Config, isExcludeRFQ bool)
}

// IUpdateL1FeeUseCase get L1 fee parameters for L2 chains and save to Redis
type IUpdateL1FeeUseCase interface {
	Handle(ctx context.Context) error
}
