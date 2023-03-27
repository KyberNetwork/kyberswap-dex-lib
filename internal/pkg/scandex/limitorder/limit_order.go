package limitorder

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/metrics"
	limitorderrepo "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository/limitorder"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

// Currently, total of reserve in limit order pool will very small with other pools. So it will filter in choosing pools process
// We will use big hardcode number to can push it into eligible pools for findRoute algorithm.
// TODO: when we has correct formula that pool's reserve can be eligible pools.
const limitOrderPoolReserve = "10000000000000000000"

type limitOrder struct {
	properties     Properties
	scanDexCfg     *config.ScanDex
	scanService    *service.ScanService
	limitOrderRepo service.ILimitOrderRepository
}

func New(scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
) (*limitOrder, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}
	return &limitOrder{
		properties:     properties,
		scanDexCfg:     scanDexCfg,
		scanService:    scanService,
		limitOrderRepo: limitorderrepo.NewHTTPClient(properties.LimitOrderHTTPUrl),
	}, nil
}

func (l *limitOrder) InitPool(ctx context.Context) error {
	return nil
}

func (l *limitOrder) UpdateNewPools(ctx context.Context) {
	logger.Infoln("Starting UpdateNewPools for LimitOrder....")
	for {
		limitOrderPairs, err := l.limitOrderRepo.ListAllPairs(ctx, valueobject.ChainID(l.scanService.Config().ChainID))
		if err != nil {
			logger.Warnf("Cannot get list supported pair cause by %v", err)
		}
		tokenPairs := l.extractTokenPairs(limitOrderPairs)
		if len(tokenPairs) == 0 {
			tokenPairs = l.properties.PredefineSupportedPairs
		}
		for _, pair := range tokenPairs {
			poolAddress := l.getPoolID(pair.Token0, pair.Token1)
			if l.scanService.ExistPool(ctx, poolAddress) {
				logger.Infof("existPool for pairs with poolAddress=%s", poolAddress)
				continue
			}
			newPool, err := l.initPool(ctx, pair)
			if err != nil {
				logger.Warnf("cannot init pool for limit order pair=%v cause by %v", pair, err)
				continue
			}
			logger.Infof("saving new Pool with pool=%v", newPool)
			err = l.scanService.SavePool(ctx, newPool)
			if err != nil {
				logger.Warnf("cannot save new pool for limit order pair=%v cause by %v", pair, err)
			}

		}
		time.Sleep(time.Duration(l.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (l *limitOrder) extractTokenPairs(limitOrderPairs []*valueobject.LimitOrderPair) []*valueobject.TokenPair {
	tokenPairMapping := make(map[string]*valueobject.TokenPair, 0)
	for _, pair := range limitOrderPairs {
		tokenPair := toTokenPair(pair)
		poolID := l.getPoolID(tokenPair.Token0, tokenPair.Token1)
		if _, ok := tokenPairMapping[poolID]; !ok {
			tokenPairMapping[poolID] = tokenPair
		}
	}
	result := make([]*valueobject.TokenPair, 0, len(tokenPairMapping))
	for _, tokenPair := range tokenPairMapping {
		result = append(result, tokenPair)
	}
	return result
}

func toTokenPair(limitOrderPair *valueobject.LimitOrderPair) *valueobject.TokenPair {
	lowerMakerAsset, lowerTakerAsset := strings.ToLower(limitOrderPair.MakerAsset), strings.ToLower(limitOrderPair.TakerAsset)
	if lowerMakerAsset > lowerTakerAsset {
		return &valueobject.TokenPair{
			Token0: lowerMakerAsset,
			Token1: lowerTakerAsset,
		}
	}
	return &valueobject.TokenPair{
		Token0: lowerTakerAsset,
		Token1: lowerMakerAsset,
	}
}

func (l *limitOrder) UpdateReserves(ctx context.Context) {
	run := func() error {
		sum := int32(0)
		startTime := time.Now()
		defer func() {
			executionTime := time.Since(startTime)

			logger.
				WithFields(logger.Fields{
					"dex":               l.scanDexCfg.Id,
					"poolsUpdatedCount": sum,
					"duration":          executionTime.Milliseconds(),
				}).
				Info("finished UpdateReserves")

			metrics.HistogramScannerUpdateReservesDuration(executionTime, l.scanDexCfg.Id, int(sum))
		}()

		pools := l.scanService.GetPoolIdsByExchange(ctx, l.scanDexCfg.Id)
		var wg sync.WaitGroup
		concurrentGoroutines := make(chan struct{}, l.properties.ConcurrentBatches)

		for i := 0; i < len(pools); i += l.properties.UpdateReserveBulk {
			wg.Add(1)
			end := i + l.properties.UpdateReserveBulk
			if end > len(pools) {
				end = len(pools)
			}
			go func(s, e int) {
				defer func() {
					wg.Done()
					<-concurrentGoroutines
				}()
				concurrentGoroutines <- struct{}{}
				pools, err := l.scanService.GetPoolsByAddresses(ctx, pools[s:e])
				if err != nil {
					logger.Errorf(err.Error())
					return
				}
				count := l.updateReserves(ctx, pools)
				atomic.AddInt32(&sum, int32(count))

			}(i, end)
		}
		wg.Wait()
		logger.Infof("update reserves %v limit order pools ==== in %v", sum, time.Since(startTime))
		return nil
	}
	for {
		err := run()
		if err != nil {
			logger.Warnf("can not update reserve err=%v", err)
		}
		time.Sleep(l.properties.ReserveJobInterval.Duration)
	}
}

func (l *limitOrder) updateReserves(ctx context.Context, pools []entity.Pool) int {
	updatedPools := 0
	for _, pool := range pools {
		err := l.updatePoolExtra(ctx, pool)
		if err != nil {
			logger.Warnf("error when updating pool extra for pool %s cause by %v", pool.Address, err)
		}
		err = l.updatePoolReserve(ctx, pool)
		if err != nil {
			logger.Warnf("error when updating pool reserve for pool %s cause by %v", pool.Address, err)
		}
		updatedPools++
	}
	return updatedPools
}

func (l *limitOrder) updatePoolExtra(ctx context.Context, pool entity.Pool) error {
	extra, err := l.getPoolExtra(ctx, pool)
	if err != nil {
		logger.Warnf("error when getting pool extra for pool %s cause by %v", pool.Address, err)
		return err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	return l.scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
}

func (l *limitOrder) updatePoolReserve(ctx context.Context, pool entity.Pool) error {
	// Save pool reserve
	reserves := []string{limitOrderPoolReserve, limitOrderPoolReserve}
	return l.scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), reserves)
}

func (l *limitOrder) getPoolExtra(ctx context.Context, pool entity.Pool) (Extra, error) {
	tokens := pool.Tokens
	if len(tokens) < 2 {
		return Extra{}, fmt.Errorf("number of tokens should be greater or equal than 2")
	}
	token0, token1 := tokens[0], tokens[1]
	if strings.ToLower(token0.Address) < strings.ToLower(token1.Address) {
		token0, token1 = tokens[1], tokens[0]
	}
	extra := Extra{}
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		buyOrders, err := l.limitOrderRepo.ListOrders(ctx, service.ListOrdersFilter{
			ChainID:             valueobject.ChainID(l.scanService.Config().ChainID),
			MakerAsset:          token0.Address,
			TakerAsset:          token1.Address,
			ExcludeExpiredOrder: true,
		})
		if err != nil {
			return err
		}
		extra.BuyOrders = buyOrders
		return nil
	})
	g.Go(func() error {
		sellOrders, err := l.limitOrderRepo.ListOrders(ctx, service.ListOrdersFilter{
			ChainID:             valueobject.ChainID(l.scanService.Config().ChainID),
			MakerAsset:          token1.Address,
			TakerAsset:          token0.Address,
			ExcludeExpiredOrder: true,
		})
		if err != nil {
			return err
		}
		extra.SellOrders = sellOrders
		return nil
	})
	err := g.Wait()

	return extra, err
}

func (l *limitOrder) UpdateTotalSupply(ctx context.Context) {
	// Do not nothing
}
