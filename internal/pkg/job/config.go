package job

import (
	"time"
)

type (
	Config struct {
		IndexPools              IndexPoolsJobConfig           `mapstructure:"indexPools"`
		UpdateSuggestedGasPrice UpdateSuggestedGasPriceConfig `mapstructure:"updateSuggestedGasPrice"`
	}

	IndexPoolsJobConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}

	UpdateSuggestedGasPriceConfig struct {
		Interval time.Duration `mapstructure:"interval"`
	}
)
