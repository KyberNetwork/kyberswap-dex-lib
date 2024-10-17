package config

import (
	"errors"

	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/job"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/server/http"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

var (
	ErrNoRouterAddress    = errors.New("no aggregation router address")
	ErrNoExecutorAddress  = errors.New("no aggregation executor address")
	ErrZeroAEVMFakeWallet = errors.New("zero AEVM fake wallet")
	ErrMissingAEVMConfigs = errors.New("missing AEVM configs")
)

type Config struct {
	Env            string                    `mapstructure:"env" json:"env"`
	Http           *http.HTTPConfig          `mapstructure:"http" json:"http"`
	Common         Common                    `mapstructure:"common" json:"common"`
	Log            *Log                      `mapstructure:"log" json:"log"`
	Redis          redis.Config              `mapstructure:"redis" json:"redis"`
	PregenRedis    redis.Config              `mapstructure:"pregenRedis" json:"pregenRedis"`
	PoolRedis      redis.Config              `mapstructure:"poolRedis" json:"poolRedis"`
	PoolEventRedis redis.Config              `mapstructure:"poolEventRedis" json:"poolEventRedis"`
	DogstatsdHost  string                    `mapstructure:"ddAgentHost" json:"ddAgentHost"`
	KeyPair        KeyPairInfo               `mapstructure:"keyPair" json:"keyPair"`
	ReloadConfig   reloadconfig.ReloadConfig `mapstructure:"reloadConfig" json:"reloadConfig"`
	SecretKey      string                    `mapstructure:"secretKey" json:"-"`
	Metrics        metrics.Config            `mapstructure:"metrics" json:"metrics"`
	GRPC           ServerListen              `mapstructure:"grpc" json:"grpc"`
	Encoder        encode.Config             `mapstructure:"encoder" json:"encoder"`
	UseCase        usecase.Config            `mapstructure:"useCase" json:"useCase"`
	Repository     repository.Config         `mapstructure:"repository" json:"repository"`
	Job            job.Config                `mapstructure:"job" json:"job"`
	Validator      validator.Config          `mapstructure:"validator" json:"validator"`
	Pprof          bool                      `mapstructure:"pprof" json:"pprof"`
	AEVMEnabled    bool                      `mapstructure:"aevmEnabled" json:"aevmEnabled"`
	AEVM           *AEVM                     `mapstructure:"aevm" json:"aevm"`

	BundledRouteEnabled bool `mapstructure:"bundledRouteEnabled" json:"bundledRouteEnabled"`
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

	if c.AEVMEnabled {
		if c.AEVM == nil {
			return ErrMissingAEVMConfigs
		}
		if utils.IsEmptyString(c.AEVM.FakeWallet) || common.HexToAddress(c.AEVM.FakeWallet) == (common.Address{}) {
			return ErrZeroAEVMFakeWallet
		}
	}

	return nil
}
