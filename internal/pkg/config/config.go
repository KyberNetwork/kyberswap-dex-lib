package config

import (
	"errors"

	"github.com/KyberNetwork/router-service/internal/pkg/job"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/server/http"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

var (
	ErrNoRouterAddress   = errors.New("no aggregation router address")
	ErrNoExecutorAddress = errors.New("no aggregation executor address")
)

type Config struct {
	Env           string                    `mapstructure:"env" json:"env"`
	Http          *http.HTTPConfig          `mapstructure:"http" json:"http"`
	Common        Common                    `mapstructure:"common" json:"common"`
	Log           *Log                      `mapstructure:"log" json:"log"`
	Redis         redis.Config              `mapstructure:"redis" json:"redis"`
	PoolRedis     redis.Config              `mapstructure:"poolRedis" json:"poolRedis"`
	DogstatsdHost string                    `mapstructure:"ddAgentHost" json:"ddAgentHost"`
	KeyPair       KeyPairInfo               `mapstructure:"keyPair" json:"keyPair"`
	ReloadConfig  reloadconfig.ReloadConfig `mapstructure:"reloadConfig" json:"reloadConfig"`
	SecretKey     string                    `mapstructure:"secretKey" json:"secretKey"`
	Metrics       metrics.Config            `mapstructure:"metrics" json:"metrics"`
	GRPC          ServerListen              `mapstructure:"grpc" json:"grpc"`
	Encoder       encode.Config             `mapstructure:"encoder" json:"encoder"`
	UseCase       usecase.Config            `mapstructure:"useCase" json:"useCase"`
	Repository    repository.Config         `mapstructure:"repository" json:"repository"`
	Job           job.Config                `mapstructure:"job" json:"job"`
}

func (c *Config) Validate() error {
	if utils.IsEmptyString(c.Encoder.RouterAddress) {
		return ErrNoRouterAddress
	}

	if utils.IsEmptyString(c.Encoder.ExecutorAddress) {
		return ErrNoExecutorAddress
	}

	if utils.IsEmptyString(c.UseCase.GetRoute.RouterAddress) {
		return ErrNoRouterAddress
	}

	return nil
}
