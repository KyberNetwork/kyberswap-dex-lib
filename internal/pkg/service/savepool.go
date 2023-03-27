package service

import (
	"context"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type savePool struct {
	scan     *ScanService
	poolRepo repository.IPoolRepository
}

func NewSavePool(
	scan *ScanService,
	poolRepo repository.IPoolRepository,
) *savePool {
	return &savePool{
		scan:     scan,
		poolRepo: poolRepo,
	}
}

func (t *savePool) UpdateData(ctx context.Context) {
	run := func() error {
		start := time.Now()
		updatePriceTime := time.Since(start)
		count := 0
		chunks := utils.Chunks(t.poolRepo.Keys(ctx), 1000)
		for _, ids := range chunks {
			for _, id := range ids {
				pool, err := t.poolRepo.Get(ctx, id)
				if err != nil {
					continue
				}
				reserveUsd, err := t.scan.calculateReserveUsd(ctx, pool)

				if err != nil {
					continue
				}

				amplifiedTvl, err := t.scan.calculateAmplifiedTvl(ctx, pool)

				if err != nil {
					continue
				}

				if pool.ReserveUsd != reserveUsd || pool.AmplifiedTvl != amplifiedTvl {
					pool.ReserveUsd = reserveUsd
					pool.AmplifiedTvl = amplifiedTvl

					count = count + 1
					if err := t.poolRepo.Persist(ctx, pool); err != nil {
						logger.Errorf("failed to persist pool %s, err: %v", pool.Address, err)
						continue
					}

					if err := t.scan.indexPair(ctx, pool); err != nil {
						logger.Errorf("failed to index pair for pool %s, err: %v", pool.Address, err)
						continue
					}
				}
			}
		}

		logger.Infof("Save %d/%d pool in %v", count, t.poolRepo.Count(ctx), updatePriceTime)

		return nil
	}

	for {
		err := run()
		time.Sleep(10 * time.Minute)
		if err != nil {
			logger.Errorf("failed to update err=%v", err)
		}

	}
}
