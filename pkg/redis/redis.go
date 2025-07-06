package redis

import (
	"context"

	"github.com/KyberNetwork/service-framework/pkg/client"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Redis struct {
	config *Config
	Client redis.UniversalClient
}

func New(cfg *Config) (*Redis, error) {
	rdb := client.NewRedisClient(context.Background(), &redis.UniversalOptions{
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
	if clusterClient, ok := universalClient.(*redis.ClusterClient); ok {
		opts := clusterClient.Options()
		log.Debug().
			Int("clusterClientOpts.PoolSize", opts.PoolSize).
			Int("clusterClientOpts.MinIdleConns", opts.MinIdleConns).
			Dur("clusterClientOpts.PoolTimeout", opts.PoolTimeout).
			Dur("clusterClientOpts.ConnMaxIdleTime", opts.ConnMaxIdleTime).
			Dur("clusterClientOpts.DialTimeout", opts.DialTimeout).
			Dur("clusterClientOpts.ReadTimeout", opts.ReadTimeout).
			Dur("clusterClientOpts.WriteTimeout", opts.WriteTimeout).
			Msg("New Redis")
	} else if cli, ok := universalClient.(*redis.Client); ok {
		opts := cli.Options()
		log.Debug().
			Int("client.PoolSize", opts.PoolSize).
			Int("client.MinIdleConns", opts.MinIdleConns).
			Dur("client.PoolTimeout", opts.PoolTimeout).
			Dur("client.ConnMaxIdleTime", opts.ConnMaxIdleTime).
			Dur("client.DialTimeout", opts.DialTimeout).
			Dur("client.ReadTimeout", opts.ReadTimeout).
			Dur("client.WriteTimeout", opts.WriteTimeout).
			Msg("New Redis")
	}
}
