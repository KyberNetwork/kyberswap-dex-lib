package repository

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type PriceDatastoreRedisRepository struct {
	db *redis.Redis
}

func NewPriceDataStoreRedisRepository(
	db *redis.Redis,
) *PriceDatastoreRedisRepository {
	return &PriceDatastoreRedisRepository{
		db: db,
	}
}

func (r *PriceDatastoreRedisRepository) FindAll(
	ctx context.Context,
) ([]entity.Price, error) {
	priceMap, err := r.db.Client.HGetAll(
		ctx,
		r.db.FormatKey(entity.PriceKey),
	).Result()
	if err != nil {
		return nil, err
	}

	prices := make([]entity.Price, 0, len(priceMap))

	for key, priceString := range priceMap {
		prices = append(prices, entity.DecodePrice(key, priceString))
	}

	return prices, nil
}

func (r *PriceDatastoreRedisRepository) FindByAddresses(
	ctx context.Context,
	addresses []string,
) ([]entity.Price, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	priceStrings, err := r.db.Client.HMGet(
		ctx,
		r.db.FormatKey(entity.PriceKey),
		addresses...,
	).Result()
	if err != nil {
		return nil, err
	}

	prices := make([]entity.Price, 0, len(priceStrings))

	for i, poolString := range priceStrings {
		if poolString != nil {
			prices = append(
				prices,
				entity.DecodePrice(addresses[i], poolString.(string)),
			)
		}
	}

	return prices, nil
}

func (r *PriceDatastoreRedisRepository) FindMapPriceByAddresses(
	ctx context.Context,
	addresses []string,
) (map[string]float64, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	priceStrings, err := r.db.Client.HMGet(
		ctx,
		r.db.FormatKey(entity.PriceKey),
		addresses...,
	).Result()

	if err != nil {
		return nil, err
	}

	priceByAddress := make(map[string]float64, len(priceStrings))

	for i, poolString := range priceStrings {
		if poolString != nil {
			price := entity.DecodePrice(addresses[i], poolString.(string))
			// If MarketPrice exists, we use it for accuracy. If not, we use our calculated price
			if price.MarketPrice > 0 {
				priceByAddress[price.Address] = price.MarketPrice
			} else {
				priceByAddress[price.Address] = price.Price
			}
		}

	}
	return priceByAddress, nil
}

func (r *PriceDatastoreRedisRepository) Persist(
	ctx context.Context,
	price entity.Price,
) error {
	_, err := r.db.Client.HSet(
		ctx,
		r.db.FormatKey(entity.PriceKey),
		price.Address,
		price.Encode(),
	).Result()

	return err
}

func (r *PriceDatastoreRedisRepository) Delete(
	ctx context.Context,
	price entity.Price,
) error {
	_, err := r.db.Client.HDel(
		ctx,
		r.db.FormatKey(entity.PriceKey),
		price.Address,
	).Result()

	return err
}

func (r *PriceDatastoreRedisRepository) DeleteMultiple(
	ctx context.Context,
	prices []entity.Price,
) error {
	var deleteKeys []string

	for _, p := range prices {
		deleteKeys = append(deleteKeys, p.Address)
	}

	_, err := r.db.Client.HDel(
		ctx,
		r.db.FormatKey(entity.PriceKey),
		deleteKeys...,
	).Result()

	return err
}
