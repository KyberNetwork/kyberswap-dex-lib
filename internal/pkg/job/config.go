package job

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
)

type (
	Config struct {
		IndexPools               IndexPoolsJobConfig               `mapstructure:"indexPools"`
		UpdateSuggestedGasPrice  UpdateSuggestedGasPriceConfig     `mapstructure:"updateSuggestedGasPrice"`
		GenerateBestPaths        GenerateBestPathsJobConfig        `mapstructure:"generateBestPaths"`
		TrackExecutorBalance     TrackExecutorBalanceConfig        `mapstructure:"trackExecutorBalance"`
		UpdateL1Fee              UpdateL1FeeConfig                 `mapstructure:"updateL1Fee"`
		LiquidityScoreIndexPools LiquidityScoreIndexPoolsJobConfig `mapstructure:"liquidityScoreIndexPools"`
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
		Interval time.Duration `mapstructure:"interval" default:"0s"`
	}

	LiquidityScoreIndexPoolsJobConfig struct {
		Env                         string        `mapstructure:"env" json:"env"`
		BatchSize                   time.Duration `mapstructure:"batchSize"`
		SuccessedFileName           string        `mapstructure:"successedFileName"`
		FailedFileName              string        `mapstructure:"failedFileName"`
		LiquidityScoreInputFileName string        `mapstructure:"liquidityScoreInputFileName"`
		LiquidityScoreCalcScript    string        `mapstructure:"liquidityScoreCalcScript"`
		Interval                    time.Duration `mapstructure:"interval"`
		ExportFailedTrade           bool          `mapstructure:"exportFailedTrade"`
		TargetFactorEntropy         float64       `mapstructure:"targetFactorEntropy"`
		ExportZeroScores            bool          `mapstructure:"exportZeroScores"`

		PoolEvent struct {
			ConsumerConfig consumer.Config `mapstructure:"consumerConfig"`
			BatchRate      time.Duration   `mapstructure:"batchRate"`
			BatchSize      int             `mapstructure:"batchSize"`
			RetryInterval  time.Duration   `mapstructure:"retryInterval"`
		} `mapstructure:"poolEvent"`
	}
)
