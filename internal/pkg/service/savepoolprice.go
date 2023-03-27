package service

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type savePoolPrice struct {
	scan      *ScanService
	poolRepo  repository.IPoolRepository
	priceRepo repository.IPriceRepository
}

func NewSavePoolPrice(
	scan *ScanService,
	poolRepo repository.IPoolRepository,
	priceRepo repository.IPriceRepository,
) *savePoolPrice {
	return &savePoolPrice{
		scan:      scan,
		poolRepo:  poolRepo,
		priceRepo: priceRepo,
	}
}

func (t *savePoolPrice) UpdateData(ctx context.Context) {
	run := func() error {
		start := time.Now()
		count := 0
		chunks := utils.Chunks(t.poolRepo.Keys(ctx), 1000)

		for _, ids := range chunks {
			for _, id := range ids {
				pool, err := t.poolRepo.Get(ctx, id)
				if err != nil || pool.TotalSupply == "" || pool.TotalSupply == "<nil>" {
					continue
				}
				reserveUsd, err := t.scan.calculateReserveUsd(ctx, pool)

				if err != nil {
					logger.Errorf("failed to calculate reserveUsd %v", err)
					continue
				}

				totalSupplyBF, _ := new(big.Float).SetString(pool.TotalSupply)
				totalSupply, _ := new(big.Float).Quo(totalSupplyBF, constant.TenPowDecimals(18)).Float64()
				if reserveUsd > 0 && totalSupply > 0 {
					price := entity.Price{
						Address:   pool.GetLpToken(),
						Price:     reserveUsd / totalSupply,
						Liquidity: reserveUsd,
						LpAddress: pool.Address,
					}

					if err := t.priceRepo.Set(ctx, price.Address, price); err != nil {
						logger.Errorf("failed to set price, err: %v", err)
						continue
					}

					count++
				}

			}

			return nil
		}

		logger.Infof("Save %d/%d pool price in %v", count, t.poolRepo.Count(ctx), time.Since(start))
		return nil
	}

	for {
		time.Sleep(10 * time.Second)
		err := run()
		if err != nil {
			logger.Errorf("failed to update err=%v", err)
		}
	}
}
