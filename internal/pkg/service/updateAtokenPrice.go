package service

import (
	"context"
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type updateAtokenPrice struct {
	tokenRepo repository.ITokenRepository
	priceRepo repository.IPriceRepository
}

func NewUpdateAtokenPrice(
	tokenRepo repository.ITokenRepository,
	priceRepo repository.IPriceRepository,
) *updateAtokenPrice {
	return &updateAtokenPrice{
		tokenRepo: tokenRepo,
		priceRepo: priceRepo,
	}
}

func (t *updateAtokenPrice) UpdateData(ctx context.Context) {
	run := func() error {
		start := time.Now()
		updatePriceTime := time.Since(start)
		count := 0

		tokenIds := t.tokenRepo.Keys(ctx)
		tokens, err := t.tokenRepo.GetByAddresses(ctx, tokenIds)

		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Type == "aave" {
				underlyingPrice, err := t.priceRepo.Get(ctx, token.PoolAddress)
				if err == nil {
					price := entity.Price{
						Address:   token.Address,
						Price:     underlyingPrice.Price,
						Liquidity: 9999999999999,
						LpAddress: token.Address,
					}
					err := t.priceRepo.Set(ctx, price.Address, price)
					if err != nil {
						logger.Errorf("failed to set price, err: %v", err)
						continue
					}

					count++
				}

			}
		}

		logger.Infof("save %d price in %v", count, updatePriceTime)

		return nil
	}

	for {
		err := run()
		if err != nil {
			logger.Errorf("failed to update err=%v", err)
		}
		time.Sleep(120 * time.Second)

	}
}
