package redis

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Redis struct {
	config *Config
	Client redis.UniversalClient
}

func New(cfg *Config) (*Redis, error) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:         cfg.Addresses,
		DB:            cfg.DBNumber,
		Password:      cfg.Password,
		MasterName:    cfg.MasterName,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,
		ReadOnly:      cfg.ReadOnly,
		RouteRandomly: cfg.RouteRandomly,
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	logRedisOption(rdb)
	return &Redis{config: cfg, Client: rdb}, nil
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

func logRedisOption(universalClient redis.UniversalClient) {
	clusterClient, ok := universalClient.(*redis.ClusterClient)

	if ok {
		logger.WithFieldsNonContext(logger.Fields{
			"clusterClientOpts.PoolSize":        clusterClient.Options().PoolSize,
			"clusterClientOpts.MinIdleConns":    clusterClient.Options().MinIdleConns,
			"clusterClientOpts.PoolTimeout":     clusterClient.Options().PoolTimeout,
			"clusterClientOpts.ConnMaxIdleTime": clusterClient.Options().ConnMaxIdleTime,
			"clusterClientOpts.DialTimeout":     clusterClient.Options().DialTimeout,
			"clusterClientOpts.ReadTimeout":     clusterClient.Options().ReadTimeout,
			"clusterClientOpts.WriteTimeout":    clusterClient.Options().WriteTimeout,
		}).Info("New Redis")
		return
	}

	client, ok := universalClient.(*redis.Client)
	if ok {
		logger.WithFieldsNonContext(logger.Fields{
			"client.PoolSize":        client.Options().PoolSize,
			"client.MinIdleConns":    client.Options().MinIdleConns,
			"client.PoolTimeout":     client.Options().PoolTimeout,
			"client.ConnMaxIdleTime": client.Options().ConnMaxIdleTime,
			"client.DialTimeout":     client.Options().DialTimeout,
			"client.ReadTimeout":     client.Options().ReadTimeout,
			"client.WriteTimeout":    client.Options().WriteTimeout,
		}).Info("New Redis")
		return
	}

}
