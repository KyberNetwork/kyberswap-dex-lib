package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

type PriceService struct {
	db     *redis.Redis
	config interface{}
}

func NewPriceService(db *redis.Redis, config *config.Common) *PriceService {
	priceService := PriceService{
		config: config,
		db:     db,
	}

	return &priceService
}
func SetupPriceRoute(db *redis.Redis, router *gin.RouterGroup) {
	router.GET(
		"/prices", func(c *gin.Context) {
			ctx := c.Request.Context()
			debug := utils.IsEnable(c.Query("debug"))
			ids := strings.Split(c.Query("ids"), ",")
			lowerIds := ids
			for i := range lowerIds {
				lowerIds[i] = strings.ToLower(lowerIds[i])
			}
			cmders, err := db.Client.Pipelined(
				ctx, func(tx redisv8.Pipeliner) error {
					tx.HMGet(c, db.FormatKey(entity.PriceKey), lowerIds...)
					return nil
				},
			)
			if err != nil {
				logger.Errorf("could not get data: %v", err)
				AbortWith500(c, "could not get data")
				return
			}
			result := map[string]map[string]interface{}{}

			priceResult := cmders[0].(*redisv8.SliceCmd).Val()
			for i, v := range priceResult {
				if v != nil && len(v.(string)) > 0 {
					result[ids[i]] = map[string]interface{}{
						"price": 0,
					}
					if priceResult[i] != nil && len(priceResult[i].(string)) > 0 {
						price := entity.DecodePrice(ids[i], priceResult[i].(string))
						// If MarketPrice exists, we use it. If not, we use our calculated price
						if price.MarketPrice > 0 {
							result[ids[i]]["price"] = price.MarketPrice
						} else {
							result[ids[i]]["price"] = price.Price
						}

						if debug {
							result[ids[i]]["liquidity"] = price.Liquidity
							result[ids[i]]["lpAddress"] = price.LpAddress
						}
					}
				}
			}
			RespondWith(c, http.StatusOK, "success", result)
		},
	)
}
func (t *PriceService) UpdateData() {
}
func (t *PriceService) NeedUpdate() bool {
	return false
}

func (t *PriceService) FindAll(ctx context.Context) ([]entity.Price, error) {
	priceMap, err := t.db.Client.HGetAll(ctx, t.db.FormatKey(entity.PriceKey)).Result()
	if err != nil {
		return nil, err
	}
	prices := make([]entity.Price, 0, len(priceMap))
	for key, priceString := range priceMap {
		prices = append(prices, entity.DecodePrice(key, priceString))
	}
	return prices, nil
}
func (t *PriceService) FindByAddress(ctx context.Context, addresses []string) ([]entity.Price, error) {

	if len(addresses) == 0 {
		return nil, nil
	}
	priceStrings, err := t.db.Client.HMGet(ctx, t.db.FormatKey(entity.PriceKey), addresses...).Result()
	if err != nil {
		return nil, err
	}
	prices := make([]entity.Price, 0, len(priceStrings))
	for i, poolString := range priceStrings {
		if poolString != nil {
			prices = append(prices, entity.DecodePrice(addresses[i], poolString.(string)))
		}

	}
	return prices, nil
}
func (t *PriceService) FindMapPriceByAddress(ctx context.Context, addresses []string) (map[string]entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "FindMapPriceByAddress")
	defer span.Finish()
	if len(addresses) == 0 {
		return nil, nil
	}
	priceStrings, err := t.db.Client.HMGet(ctx, t.db.FormatKey(entity.PriceKey), addresses...).Result()
	if err != nil {
		return nil, err
	}
	prices := make(map[string]entity.Price, len(priceStrings))
	for i, poolString := range priceStrings {
		if poolString != nil {
			price := entity.DecodePrice(addresses[i], poolString.(string))
			prices[price.Address] = price
		}

	}
	return prices, nil
}
