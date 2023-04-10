package redis

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Redis struct {
	config *Config
	Client redis.UniversalClient
}

func New(cfg *Config) (*Redis, error) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        cfg.Addresses,
		DB:           cfg.DBNumber,
		Password:     cfg.Password,
		MasterName:   cfg.MasterName,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

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
