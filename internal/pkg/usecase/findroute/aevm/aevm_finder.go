package aevm

import (
	"bytes"
	"context"
	"fmt"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	findrouteencode "github.com/KyberNetwork/aevm/usecase/findroute/encode"
	aevmpool "github.com/KyberNetwork/aevm/usecase/pool/common"
	dexlibmsgpack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	pkg_source_kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	pkg_source_limitorder "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pkg_source_synthetix "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/msgpack/v5"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func mustNotError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_kyberpmm.Inventory{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_limitorder.Inventory{}))
	mustNotError(msgpack.RegisterConcreteType(&pkg_source_synthetix.AtomicLimits{}))
}

// AEVMFinder depending on configurations, AEVMFinder performs routes finding localy or in remote AEVM server.
type AEVMFinder struct {
	aevmClient     aevmclient.Client
	baseFinder     findroute.IFinder
	poolsPublisher IPoolsPublisher
	opts           valueobject.FinderOptions
}

func NewAEVMFinder(baseFinder findroute.IFinder, aevmClient aevmclient.Client, poolsPublisher IPoolsPublisher, opts valueobject.FinderOptions) *AEVMFinder {
	return &AEVMFinder{
		aevmClient:     aevmClient,
		baseFinder:     baseFinder,
		poolsPublisher: poolsPublisher,
		opts:           opts,
	}
}

func (f *AEVMFinder) Find(ctx context.Context, input findroute.Input, data findroute.FinderData) ([]*valueobject.Route, error) {
	if !f.opts.UseAEVMRemoteFinder {
		data.PoolBucket = shallowClonePoolsBucket(data.PoolBucket)

		useAEVMPool := f.opts.LocalUseAEVMPool
		for _, poolsMap := range []map[string]poolpkg.IPoolSimulator{
			data.PoolBucket.PerRequestPoolsByAddress,
			data.PoolBucket.ChangedPools,
		} {
			for _, pool := range poolsMap {
				if aevmPool, ok := pool.(aevmpool.IAEVMPool); ok {
					if useAEVMPool {
						aevmPool.UseAsAEVMPool(f.aevmClient)
					} else {
						aevmPool.UseAsNormalPool()
					}
				}
			}
		}

		return f.baseFinder.Find(ctx, input, data)
	}

	start := time.Now()

	prepared, err := f.baseFinder.Prepare(ctx, input, data)
	if err != nil {
		logger.Warnf(ctx, "could not Prepare() base finder: %s", err)
		return nil, fmt.Errorf("could not Prepare() base finder: %w", err)
	}

	data.UseAEVMPool = f.opts.RemoteUseAEVMPool
	data.PoolBucket = shallowClonePoolsBucket(data.PoolBucket)
	// Remove IPoolSimulators which already published under `data.PublishedPoolsStorageID` from `data.PoolBucket`.
	// The remote IFinder will fill in the removed IPoolSimulators using its published pools.
	removePublishedPoolsFromPoolsBucket(data.PoolBucket, f.poolsPublisher.PublishedPoolIDs(data.PublishedPoolsStorageID))

	params, err := findrouteencode.EncodeFindRouteParams(prepared, &input, &data)
	if err != nil {
		logger.Warnf(ctx, "could not EncodeFindRouteParams: %s", err)
		return nil, fmt.Errorf("could not EncodeFindRouteParams: %w", err)
	}

	result, err := f.aevmClient.FindRoute(ctx, params)
	if err != nil {
		logger.Warnf(ctx, "FindRoute return error: %s", err)
		return nil, fmt.Errorf("FindRoute return error: %w", err)
	}

	routes := new([]*valueobject.Route)
	routesDec := dexlibmsgpack.NewDecoder(bytes.NewReader(result.EncodedRoutes))
	if err := routesDec.Decode(routes); err != nil {
		logger.Warnf(ctx, "could not decode valueobject.Routes: %s", err)
		return nil, fmt.Errorf("could not decode valueobject.Routes: %w", err)
	}

	took := time.Since(start)
	logger.Infof(ctx, "AEVMFinder.Find() took %s", took.String())

	return *routes, nil
}

func (f *AEVMFinder) Prepare(_ context.Context, _ findroute.Input, _ findroute.FinderData) (findroute.IFinder, error) {
	return f, nil
}

func shallowClonePoolsBucket(bucket *valueobject.PoolBucket) *valueobject.PoolBucket {
	pools := make(map[string]poolpkg.IPoolSimulator, len(bucket.PerRequestPoolsByAddress))
	for addr, pool := range bucket.PerRequestPoolsByAddress {
		pools[addr] = pool
	}
	changedPools := make(map[string]poolpkg.IPoolSimulator, len(bucket.ChangedPools))
	for addr, pool := range bucket.ChangedPools {
		changedPools[addr] = pool
	}
	return &valueobject.PoolBucket{
		PerRequestPoolsByAddress: pools,
		ChangedPools:             changedPools,
	}
}

func removePublishedPoolsFromPoolsBucket(bucket *valueobject.PoolBucket, poolIDsToRemove map[string]struct{}) {
	var (
		removeCounter int
	)
	for poolID := range bucket.PerRequestPoolsByAddress {
		if _, ok := poolIDsToRemove[poolID]; ok {
			removeCounter++
			bucket.PerRequestPoolsByAddress[poolID] = nil
		}
	}
	for poolID := range bucket.ChangedPools {
		if _, ok := poolIDsToRemove[poolID]; ok {
			removeCounter++
			bucket.ChangedPools[poolID] = nil
		}
	}
}
