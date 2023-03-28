package service

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

const StatsKey = "ScanBlock"
const TotalPoolKey = "totalPools"
const TotalTokenKey = "totalTokens"

type StatsService struct {
	scan      *ScanService
	poolRepo  repository.IPoolRepository
	statsRepo repository.IStatsRepository
}

func NewStats(
	scan *ScanService,
	poolRepo repository.IPoolRepository,
	statsRepo repository.IStatsRepository,
) *StatsService {
	return &StatsService{
		scan:      scan,
		poolRepo:  poolRepo,
		statsRepo: statsRepo,
	}
}

type PoolStatsItem struct {
	Size   int     `json:"poolSize"`
	Tvl    float64 `json:"tvl"`
	Tokens int     `json:"tokenSize"`
}

func SetupStatsRoute(db *redis.Redis, router *gin.RouterGroup) {
	router.GET("/stats", func(c *gin.Context) {
		ctx := c.Request.Context()
		var res = map[string]interface{}{}
		pipe := db.Client.TxPipeline()
		pools := pipe.HGet(ctx, db.FormatKey(StatsKey), entity.PoolKey)
		totalPools := pipe.HGet(ctx, db.FormatKey(StatsKey), TotalPoolKey)
		totalTokens := pipe.HGet(ctx, db.FormatKey(StatsKey), TotalTokenKey)

		_, err := pipe.Exec(ctx)
		if err != nil {
			logger.Errorf("could not get data with key=%s and field=%s,%s,%s: %v", db.FormatKey(StatsKey), entity.PoolKey, TotalPoolKey, TotalTokenKey, err)
			AbortWith500(c, "could not get data")
			return
		}

		poolStats := map[string]PoolStatsItem{}
		err = db.Decode([]byte(pools.Val()), &poolStats)
		if err != nil {
			logger.Errorf("could not decode data: %v", err)
			AbortWith500(c, "could not decode data")
			return
		}
		res["pools"] = poolStats
		res[TotalPoolKey] = totalPools.Val()
		res[TotalTokenKey] = totalTokens.Val()
		RespondWith(c, http.StatusOK, "success", res)
	})
}

func (t *StatsService) UpdateData(ctx context.Context) {
	run := func() error {
		start := time.Now()
		poolStatsItems := map[string]entity.PoolStatsItem{}
		ksTokenMap := make(map[string]interface{})
		totalTokensMap := make(map[string]bool)

		poolIds := t.poolRepo.Keys(ctx)
		pools, err := t.poolRepo.GetByAddresses(ctx, poolIds)

		if err != nil {
			return err
		}

		for _, pool := range pools {
			statsItem := poolStatsItems[pool.Exchange]
			if ksTokenMap[pool.Exchange] == nil {
				ksTokenMap[pool.Exchange] = make(map[string]bool)
			}
			dexTokenMap := ksTokenMap[pool.Exchange].(map[string]bool)
			statsItem.Tokens = len(dexTokenMap)
			for _, token := range pool.Tokens {
				dexTokenMap[token.Address] = true
				totalTokensMap[token.Address] = true
			}
			statsItem.Size += 1
			statsItem.Tvl += pool.ReserveUsd

			poolStatsItems[pool.Exchange] = statsItem
		}

		stats := entity.Stats{
			Pools:       poolStatsItems,
			TotalPools:  t.poolRepo.Count(ctx),
			TotalTokens: len(totalTokensMap),
		}

		err = t.statsRepo.Persist(ctx, stats)
		if err != nil {
			logger.Errorf("failed to save stats, err: %v", err)
		}

		logger.Infof("Stats %d pool in %v", t.poolRepo.Count(ctx), time.Since(start))

		return nil
	}

	for {
		time.Sleep(10 * time.Second)

		err := run()
		if err != nil {
			logger.Errorf("failed to update status rpc")
		}
	}
}
