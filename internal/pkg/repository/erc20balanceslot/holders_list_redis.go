package erc20balanceslot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

var (
	ErrNotFound = errors.New("not found")
)

type HoldersListRedisRepositoryWithCache struct {
	redisClient *redis.Redis
	redisKey    string
	cache       *cache.Cache // common.Address.String() => *entity.ERC20HoldersList
	getGroup    singleflight.Group
}

func NewHoldersListRedisRepositoryWithCache(redisClient *redis.Redis, ttlSec uint64) *HoldersListRedisRepositoryWithCache {
	return &HoldersListRedisRepositoryWithCache{
		redisClient: redisClient,
		redisKey:    redisClient.FormatKey(KeyHoldersList),
		cache:       cache.New(time.Duration(ttlSec)*time.Second, 10*time.Minute),
	}
}

func (r *HoldersListRedisRepositoryWithCache) Get(ctx context.Context, token common.Address) (*types.ERC20HoldersList, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] HoldersListRedisRepository.Get")
	defer span.End()

	key := strings.ToLower(token.String())
	if raw, ok := r.cache.Get(key); ok {
		if cached, ok := raw.(*types.ERC20HoldersList); ok {
			return cached, nil
		}
	}

	// to prevent Redis stampede
	_rawResult, _, _ := r.getGroup.Do(key, func() (interface{}, error) {
		return r.redisClient.Client.HGet(ctx, r.redisKey, key).Val(), nil
	})
	rawResult := _rawResult.(string)
	if rawResult == "" {
		return nil, ErrNotFound
	}

	result := new(types.ERC20HoldersList)
	if err := json.Unmarshal([]byte(rawResult), result); err != nil {
		return nil, fmt.Errorf("could not unmarshal entity.ERC20HoldersList token %v err %v", token, err)
	}
	r.cache.Set(key, result, cache.DefaultExpiration)

	return result, nil
}

type WatchlistRedisRepository struct {
	redisClient *redis.Redis
}

func NewWatchlistRedisRepository(redisClient *redis.Redis) *WatchlistRedisRepository {
	return &WatchlistRedisRepository{
		redisClient: redisClient,
	}
}

func (r *WatchlistRedisRepository) Notify(ctx context.Context, token common.Address) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "[erc20balanceslot] WatchlistRedisRepository.Add")
	defer span.End()

	r.redisClient.Client.Publish(ctx, r.redisClient.FormatKey(KeyWatchlistNotify), strings.ToLower(token.String())).Val()
	return nil
}
