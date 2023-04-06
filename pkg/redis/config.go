package redis

type Config struct {
	MasterName   string   `mapstructure:"masterName" json:"masterName" default:""`
	Addresses    []string `mapstructure:"addresses" json:"addresses" default:""`
	DBNumber     int      `mapstructure:"dbNumber" json:"dbNumber" default:"0"`
	Prefix       string   `mapstructure:"prefix" json:"prefix" default:""`
	Password     string   `mapstructure:"password" json:"-" default:""`
	ReadTimeout  int      `mapstructure:"readTimeout" json:"readTimeout" default:"0"`
	WriteTimeout int      `mapstructure:"writeTimeout" json:"writeTimeout" default:"0"`
}
