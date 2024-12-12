package indexpools

import (
	"context"

	mapset "github.com/deckarep/golang-set/v2"
)

func NewBlacklistPoolIndex(repository IBlacklistIndexPoolRepository) *BlacklistPoolIndex {
	return &BlacklistPoolIndex{
		repository: repository,
	}
}

func (uc *BlacklistPoolIndex) GetBlacklistIndexPools(ctx context.Context) mapset.Set[string] {
	return uc.repository.GetBlacklistIndexPools(ctx)

}

func (uc *BlacklistPoolIndex) AddToBlacklistIndexPools(ctx context.Context, addresses []string) {
	uc.repository.AddToBlacklistIndexPools(ctx, addresses)
}
