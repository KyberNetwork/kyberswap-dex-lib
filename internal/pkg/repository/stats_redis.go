package repository

import (
	"context"
	"strconv"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

const StatsKey = "stats"
const PoolsKey = "pools"
const TotalPoolKey = "totalPools"
const TotalTokenKey = "totalTokens"

type StatsRedisRepository struct {
	db *redis.Redis
}

func NewStatsRedisRepository(db *redis.Redis) *StatsRedisRepository {
	return &StatsRedisRepository{
		db: db,
	}
}

func (r *StatsRedisRepository) Get(ctx context.Context) (entity.Stats, error) {
	pipe := r.db.Client.TxPipeline()

	pools := pipe.HGet(ctx, r.db.FormatKey(StatsKey), PoolsKey)
	totalPools := pipe.HGet(ctx, r.db.FormatKey(StatsKey), TotalPoolKey)
	totalTokens := pipe.HGet(ctx, r.db.FormatKey(StatsKey), TotalTokenKey)

	_, err := pipe.Exec(ctx)

	if err != nil {
		return entity.Stats{}, err
	}

	poolStats := map[string]entity.PoolStatsItem{}
	err = r.db.Decode([]byte(pools.Val()), &poolStats)
	if err != nil {
		return entity.Stats{}, err
	}

	var res entity.Stats
	res.Pools = poolStats

	if totalPoolsInt, err := strconv.Atoi(totalPools.Val()); err == nil {
		res.TotalPools = totalPoolsInt
	} else {
		res.TotalPools = 0
	}

	if totalTokensInt, err := strconv.Atoi(totalTokens.Val()); err == nil {
		res.TotalTokens = totalTokensInt
	} else {
		res.TotalTokens = 0
	}

	return res, nil
}

func (r *StatsRedisRepository) Persist(ctx context.Context, stats entity.Stats) error {
	encodedPools, err := r.db.Encode(stats.Pools)

	if err != nil {
		return err
	}

	_, err = r.db.Client.Pipelined(
		ctx, func(tx redisv9.Pipeliner) error {
			tx.HSet(ctx, r.db.FormatKey(StatsKey), PoolsKey, encodedPools)
			tx.HSet(ctx, r.db.FormatKey(StatsKey), TotalPoolKey, stats.TotalPools)
			tx.HSet(ctx, r.db.FormatKey(StatsKey), TotalTokenKey, stats.TotalTokens)
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}
