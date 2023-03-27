package redis

type Config struct {
	Host         string `mapstructure:"host" json:"host" default:"localhost"`
	Port         int    `mapstructure:"port" json:"port" default:"6379"`
	DBNumber     int    `mapstructure:"dbNumber" json:"dbNumber" default:"0"`
	Password     string `mapstructure:"password" json:"-" default:""`
	Prefix       string `mapstructure:"prefix" json:"prefix" default:""`
	ReadTimeout  int    `mapstructure:"readTimeout" json:"readTimeout" default:"0"`
	WriteTimeout int    `mapstructure:"writeTimeout" json:"writeTimeout" default:"0"`
}

type SentinelConfig struct {
	MasterName   string `mapstructure:"masterName" json:"masterName" default:""`
	SentinelPort int    `mapstructure:"sentinelPort" json:"sentinelPort" default:"26379"`
	DBNumber     int    `mapstructure:"dbNumber" json:"dbNumber" default:"0"`
	Password     string `mapstructure:"password" json:"-" default:""`
	Prefix       string `mapstructure:"prefix" json:"prefix" default:""`
	ReadTimeout  int    `mapstructure:"readTimeout" json:"readTimeout" default:"0"`
	WriteTimeout int    `mapstructure:"writeTimeout" json:"writeTimeout" default:"0"`
}
