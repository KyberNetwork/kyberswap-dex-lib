package aevm

import (
	"context"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmpool "github.com/KyberNetwork/aevm/usecase/pool/common"
	finderEntity "github.com/KyberNetwork/pathfinder-lib/pkg/entity"
	finderEngine "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	finderCommon "github.com/KyberNetwork/pathfinder-lib/pkg/finderengine/common"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type AEVMLocalFinder struct {
	aevmClient aevmclient.Client
	baseFinder finderEngine.IFinder
	opts       valueobject.FinderOptions
}

func NewAEVMLocalFinder(baseFinder finderEngine.IFinder, aevmClient aevmclient.Client, opts valueobject.FinderOptions) *AEVMLocalFinder {
	return &AEVMLocalFinder{
		aevmClient: aevmClient,
		baseFinder: baseFinder,
		opts:       opts,
	}
}

func (f *AEVMLocalFinder) Find(ctx context.Context, params finderEntity.FinderParams) ([]*finderCommon.ConstructRoute, error) {
	useAEVMPool := f.opts.LocalUseAEVMPool

	if !useAEVMPool {
		return f.baseFinder.Find(ctx, params)
	}

	for _, pool := range params.Pools {
		if aevmPool, ok := pool.(aevmpool.IAEVMPool); ok {
			if useAEVMPool {
				aevmPool.UseAsAEVMPool(f.aevmClient)
			} else {
				aevmPool.UseAsNormalPool()
			}
		}
	}

	return f.baseFinder.Find(ctx, params)
}
