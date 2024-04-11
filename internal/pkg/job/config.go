package job

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
)

type (
	Config struct {
		IndexPools              IndexPoolsJobConfig           `mapstructure:"indexPools"`
		UpdateSuggestedGasPrice UpdateSuggestedGasPriceConfig `mapstructure:"updateSuggestedGasPrice"`
		GenerateBestPaths       GenerateBestPathsJobConfig    `mapstructure:"generateBestPaths"`
		TrackExecutorBalance    TrackExecutorBalanceConfig    `mapstructure:"trackExecutorBalance"`
		UpdateL1Fee             UpdateL1FeeConfig             `mapstructure:"updateL1Fee"`
	}

	IndexPoolsJobConfig struct {
		Interval  time.Duration `mapstructure:"interval"`
		PoolEvent struct {
			ConsumerConfig consumer.Config `mapstructure:"consumerConfig"`
			BatchRate      time.Duration   `mapstructure:"batchRate"`
			BatchSize      int             `mapstructure:"batchSize"`
			RetryInterval  time.Duration   `mapstructure:"retryInterval"`
		} `mapstructure:"poolEvent"`

		ForceScanAllEveryNth int `mapstructure:"forceScanAllEveryNth" default:"10"`
	}

	UpdateSuggestedGasPriceConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}

	GenerateBestPathsJobConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}

	TrackExecutorBalanceConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}

	UpdateL1FeeConfig struct {
		Interval      time.Duration `mapstructure:"interval" default:"0s"`
		OracleAddress string        `mapstructure:"oracle_address"`
	}
)
