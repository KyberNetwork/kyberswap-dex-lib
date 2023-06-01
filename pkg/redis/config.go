package redis

import "time"

type Config struct {
	MasterName    string        `mapstructure:"masterName" json:"masterName" default:""`
	Addresses     []string      `mapstructure:"addresses" json:"addresses" default:""`
	DBNumber      int           `mapstructure:"dbNumber" json:"dbNumber" default:"0"`
	Prefix        string        `mapstructure:"prefix" json:"prefix" default:""`
	Password      string        `mapstructure:"password" json:"-" default:""`
	ReadOnly      bool          `mapstructure:"readOnly" json:"readOnly" default:""`
	RouteRandomly bool          `mapstructure:"routeRandomly" json:"routeRandomly" default:""`
	ReadTimeout   time.Duration `mapstructure:"readTimeout" json:"readTimeout" default:"0"`
	WriteTimeout  time.Duration `mapstructure:"writeTimeout" json:"writeTimeout" default:"0"`
}
