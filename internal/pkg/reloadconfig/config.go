package reloadconfig

import "time"

type ReloadConfig struct {
	HttpUrl     string        `mapstructure:"httpUrl" json:"httpUrl" default:""`
	Interval    time.Duration `mapstructure:"interval" json:"interval" default:"10s"`
	ServiceName string        `mapstructure:"serviceName" json:"serviceName" default:"aggregator"`
	ChainID     int           `mapstructure:"chainID" json:"chainID" default:"1"`
}
