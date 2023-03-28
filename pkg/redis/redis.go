package redis

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Redis struct {
	config *Config
	Client *redis.Client
}

type RedisCluster struct {
	config *SentinelConfig
	Client *redis.ClusterClient
}

func New(cfg *Config) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DBNumber,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	if _, err := rdb.Ping(gocontext.Background()).Result(); err != nil {
		return nil, err
	}

	return &Redis{config: cfg, Client: rdb}, nil
}

func NewSentinel(cfg *SentinelConfig) (*RedisCluster, error) {
	if cfg.MasterName == "" {
		return nil, nil
	}

	rdb := redis.NewFailoverClusterClient(&redis.FailoverOptions{
		Password: cfg.Password,
		DB:       cfg.DBNumber,
		SentinelAddrs: []string{
			fmt.Sprintf("%s-node-0.%s-headless.redis.svc.cluster.local:%v", cfg.MasterName, cfg.MasterName, cfg.SentinelPort),
			fmt.Sprintf("%s-node-1.%s-headless.redis.svc.cluster.local:%v", cfg.MasterName, cfg.MasterName, cfg.SentinelPort),
			fmt.Sprintf("%s-node-2.%s-headless.redis.svc.cluster.local:%v", cfg.MasterName, cfg.MasterName, cfg.SentinelPort),
		},
		MasterName:   fmt.Sprintf("%s-master", cfg.MasterName),
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	if _, err := rdb.Ping(gocontext.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisCluster{config: cfg, Client: rdb}, nil
}

func (s *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}

	k := fmt.Sprintf("%s:%s", s.config.Prefix, key)
	return s.Client.Set(ctx, k, cacheEntry, expiration).Err()
}

func (s *Redis) Get(ctx context.Context, key string, src interface{}) error {
	k := fmt.Sprintf("%s:%s", s.config.Prefix, key)
	val, err := s.Client.Get(ctx, k).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), &src)
}
func (s *Redis) Encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}
func (s *Redis) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
func (s *Redis) FormatKey(args ...interface{}) string {
	return utils.Join(s.config.Prefix, utils.Join(args...))
}
