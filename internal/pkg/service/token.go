package service

import (
	"context"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

type TokenService struct {
	db     *redis.Redis
	config *config.Common
}

func NewToken(db *redis.Redis, config *config.Common) *TokenService {
	var ret = TokenService{
		config: config,
		db:     db,
	}
	return &ret
}

func (t *TokenService) UpdateData() {
}
func (t *TokenService) NeedUpdate() bool {
	return false
}
func SetupTokenRoute(db *redis.Redis, router *gin.RouterGroup) {
	router.GET("/tokens", func(c *gin.Context) {
		ctx := c.Request.Context()
		extra := utils.IsEnable(c.Query("extra"))
		showPoolTokens := utils.IsEnable(c.Query("poolTokens"))
		ids := strings.Split(c.Query("ids"), ",")
		lowerIds := ids
		for i := range lowerIds {
			lowerIds[i] = strings.ToLower(strings.TrimSpace(lowerIds[i]))
		}
		cmders, err := db.Client.Pipelined(ctx, func(tx redisv8.Pipeliner) error {
			tx.HMGet(ctx, db.FormatKey(entity.TokenKey), lowerIds...)
			tx.HMGet(ctx, db.FormatKey(entity.PriceKey), lowerIds...)
			return nil
		})
		if err != nil {
			logger.Errorf("could not get data: %v", err)
			AbortWith500(c, "could not get data")
			return
		}
		priceResult := cmders[1].(*redisv8.SliceCmd).Val()
		var poolMap = make(map[string]entity.Pool)
		var tokenMap = make(map[string]entity.Token)
		var priceMap = make(map[string]entity.Price)
		poolLps := sets.NewString()
		for i, v := range cmders[0].(*redisv8.SliceCmd).Val() {
			if v != nil && len(v.(string)) > 0 {
				token := entity.DecodeToken(ids[i], v.(string))
				tokenMap[token.Address] = token
				if priceResult[i] != nil && len(priceResult[i].(string)) > 0 {
					price := entity.DecodePrice(ids[i], priceResult[i].(string))
					priceMap[price.Address] = price
					if len(token.PoolAddress) > 0 {
						poolLps.Insert(token.PoolAddress)
					}

				}
			}
		}
		if showPoolTokens && poolLps.Len() > 0 {
			poolLpIds := poolLps.List()
			cmders, err = db.Client.Pipelined(ctx, func(tx redisv8.Pipeliner) error {
				tx.HMGet(ctx, db.FormatKey(entity.PoolKey), poolLpIds...)
				return nil
			})
			if err != nil {
				logger.Errorf("could not get data: %v", err)
				AbortWith500(c, "could not get data")
				return
			}
			tokenSet := sets.NewString()
			for i, v := range cmders[0].(*redisv8.SliceCmd).Val() {
				if v != nil && len(v.(string)) > 0 {
					pool, err := entity.DecodePool(poolLpIds[i], v.(string))
					if err != nil {
						logger.Errorf("could decode pool: %v", err)
						AbortWith500(c, "could decode pool")
						return
					}
					poolMap[pool.Address] = pool
					for _, token := range pool.Tokens {
						if _, ok := tokenMap[token.Address]; !ok {
							tokenSet.Insert(token.Address)
						}
					}
				}
			}
			if tokenSet.Len() > 0 {
				childTokenIds := tokenSet.List()
				cmders, err = db.Client.Pipelined(ctx, func(tx redisv8.Pipeliner) error {
					tx.HMGet(ctx, db.FormatKey(entity.TokenKey), childTokenIds...)
					return nil
				})
				if err != nil {
					logger.Errorf("could not get data: %v", err)
					AbortWith500(c, "could not get data")
					return
				}
				for i, v := range cmders[0].(*redisv8.SliceCmd).Val() {
					if v != nil && len(v.(string)) > 0 {
						token := entity.DecodeToken(childTokenIds[i], v.(string))
						tokenMap[token.Address] = token
					}
				}
			}
		}
		result := map[string]map[string]interface{}{}
		for _, id := range ids {
			lowerId := strings.ToLower(id)
			if token, ok := tokenMap[lowerId]; ok {
				result[id] = map[string]interface{}{
					"name":     token.Name,
					"decimals": token.Decimals,
					"symbol":   token.Symbol,
					"type":     token.Type,
					"price":    0,
				}

				if price, ok := priceMap[lowerId]; ok {
					// If MarketPrice exists, we use it. If not, we use our calculated price
					if price.MarketPrice > 0 {
						result[id]["price"] = price.MarketPrice
					} else {
						result[id]["price"] = price.Price
					}

					if extra {
						result[id]["liquidity"] = price.Liquidity
						result[id]["lpAddress"] = price.LpAddress
					}
				}
				if showPoolTokens && len(token.PoolAddress) > 0 {
					if pool, ok := poolMap[token.PoolAddress]; ok {
						totalSupplyBF, _ := new(big.Float).SetString(pool.TotalSupply)
						totalSupply, _ := new(big.Float).Quo(totalSupplyBF, constant.TenPowDecimals(18)).Float64()
						poolResult := map[string]interface{}{
							"address":     pool.Address,
							"totalSupply": totalSupply,
							"reserveUsd":  pool.ReserveUsd,
						}

						if len(pool.Tokens) > 0 {
							var tokens = make([]map[string]interface{}, len(pool.Tokens))
							for i, poolToken := range pool.Tokens {
								token := tokenMap[poolToken.Address]
								tokens[i] = map[string]interface{}{
									"name":     token.Name,
									"decimals": token.Decimals,
									"symbol":   token.Symbol,
									"type":     token.Type,
									"weight":   poolToken.Weight,
								}
							}
							poolResult["tokens"] = tokens
						}
						result[id]["pool"] = poolResult

					}
				}

			}
		}
		RespondWith(c, http.StatusOK, "success", result)
	})
}
func (t TokenService) FindAll(ctx context.Context) ([]entity.Token, error) {
	tokenMap, err := t.db.Client.HGetAll(ctx, t.db.FormatKey(entity.TokenKey)).Result()
	if err != nil {
		return nil, err
	}
	tokens := make([]entity.Token, 0, len(tokenMap))
	for key, tokenString := range tokenMap {
		tokens = append(tokens, entity.DecodeToken(key, tokenString))
	}
	return tokens, nil
}
func (t TokenService) FindByAddress(ctx context.Context, addresses []string) ([]entity.Token, error) {

	if len(addresses) == 0 {
		return nil, nil
	}
	tokenStrings, err := t.db.Client.HMGet(ctx, t.db.FormatKey(entity.TokenKey), addresses...).Result()
	if err != nil {
		return nil, err
	}
	tokens := make([]entity.Token, 0, len(tokenStrings))
	for i, poolString := range tokenStrings {
		if poolString != nil {
			tokens = append(tokens, entity.DecodeToken(addresses[i], poolString.(string)))
		}

	}
	return tokens, nil
}

func (t TokenService) FindMapByAddress(ctx context.Context, parentSpan tracer.Span, addresses []string) (map[string]entity.Token, error) {
	span := tracer.StartSpan("FindMapByAddress", tracer.ChildOf(parentSpan.Context()))
	defer span.Finish()
	if len(addresses) == 0 {
		return nil, nil
	}
	tokenStrings, err := t.db.Client.HMGet(ctx, t.db.FormatKey(entity.TokenKey), addresses...).Result()
	if err != nil {
		return nil, err
	}
	tokenMap := map[string]entity.Token{}
	token := entity.Token{}
	for i, tokenString := range tokenStrings {
		if tokenString != nil {
			token = entity.DecodeToken(addresses[i], tokenString.(string))
			tokenMap[token.Address] = token
		}
	}
	return tokenMap, nil
}
