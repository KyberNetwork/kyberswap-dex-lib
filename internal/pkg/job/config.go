package job

import (
	"time"
)

type (
	Config struct {
		IndexPools     IndexPoolsJobConfig  `mapstructure:"indexPools"`
		UpdateGasPrice UpdateGasPriceConfig `mapstructure:"updateGasPrice"`
	}

	IndexPoolsJobConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}

	UpdateGasPriceConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}
)
