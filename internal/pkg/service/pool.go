package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type PoolService struct {
	//
	db *redis.Redis
}

func NewPool(db *redis.Redis) *PoolService {
	var ret = PoolService{
		db: db,
	}
	return &ret
}

func SetupPoolRoute(db *redis.Redis, router *gin.RouterGroup) {
	span := tracer.StartSpan("SetupPoolRoute")
	defer span.Finish()
	poolService := NewPool(db)
	router.GET("/pools", func(c *gin.Context) {
		span, _ := tracer.StartSpanFromContext(c.Request.Context(), "SetupPoolRoute")
		defer span.Finish()
		ids := strings.Split(c.Query("ids"), ",")
		lowerIds := ids
		for i := range lowerIds {
			lowerIds[i] = strings.ToLower(strings.TrimSpace(lowerIds[i]))
		}
		pools, _ := poolService.FindByAddress(c.Request.Context(), lowerIds)
		RespondWith(c, http.StatusOK, "success", pools)
	})
}
func (t PoolService) UpdateData(ctx context.Context) {

}
func (t PoolService) FindAll(ctx context.Context) ([]entity.Pool, error) {
	poolMap, err := t.db.Client.HGetAll(ctx, t.db.FormatKey(entity.PoolKey)).Result()
	if err != nil {
		return nil, err
	}
	pools := make([]entity.Pool, 0, len(poolMap))
	for key, poolString := range poolMap {
		pool, derr := entity.DecodePool(key, poolString)
		if derr != nil {
			return nil, derr
		}
		pools = append(pools, pool)
	}
	return pools, nil
}
func (t PoolService) FindByAddress(ctx context.Context, addresses []string) ([]entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "FindByAddress")
	defer span.Finish()

	if len(addresses) == 0 {
		return nil, nil
	}
	poolStrings, err := t.db.Client.HMGet(ctx, t.db.FormatKey(entity.PoolKey), addresses...).Result()
	if err != nil {
		return nil, err
	}
	pools := make([]entity.Pool, len(poolStrings))
	for i, poolString := range poolStrings {
		if poolString != nil {
			pools[i], err = entity.DecodePool(addresses[i], poolString.(string))
			if err != nil {
				return nil, err
			}
		} else {
			pools[i].Address = addresses[i]
		}
	}
	return pools, nil
}
