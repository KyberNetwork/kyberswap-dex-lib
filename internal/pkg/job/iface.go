package job

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	mapset "github.com/deckarep/golang-set/v2"
)

// IGetAllPoolAddressesUseCase get all pool addresses from Redis
type IGetAllPoolAddressesUseCase interface {
	Handle(ctx context.Context) ([]string, error)
}

// IIndexPoolsUseCase get pools info save/update into Redis sorted set, score by reserveUsd or amplifiedTvl
type IIndexPoolsUseCase interface {
	Handle(ctx context.Context, command dto.IndexPoolsCommand) *dto.IndexPoolsResult
	RemovePoolFromIndexes(ctx context.Context, pool *entity.Pool) error
}

// IUpdateSuggestedGasPriceUseCase get suggested gas price from rpc and save it to Redis
type IUpdateSuggestedGasPriceUseCase interface {
	Handle(ctx context.Context) (*dto.UpdateGasPriceResult, error)
}

// IUpdateL1FeeUseCase get L1 fee parameters for L2 chains and save to Redis
type IUpdateL1FeeUseCase interface {
	Handle(ctx context.Context) error
}

type ITradeGeneratorUsecase interface {
	Handle(ctx context.Context,
		output chan<- indexpools.TradesGenerationOutput, indexBlacklistWlPools mapset.Set[string], addresses mapset.Set[indexpools.TradesGenerationInput])
}

type IRemovePoolsFromIndexUseCase interface {
	Handle(ctx context.Context, indexBlacklistWlPools mapset.Set[string]) error
}

type IUpdatePoolScores interface {
	Handle(ctx context.Context, scoresFileName string) error
}

type IBlacklistIndexPoolsUsecase interface {
	GetBlacklistIndexPools(ctx context.Context) mapset.Set[string]
	AddToBlacklistIndexPools(ctx context.Context, addresses []string)
}

type IRemovePoolIndexUseCase interface {
	RemovePoolAddressFromLiqScoreIndexes(ctx context.Context, addresses ...string) error
}
