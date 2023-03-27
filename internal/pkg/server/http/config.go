package http

type HTTPConfig struct {
	BindAddress string `mapstructure:"bindAddress" default:"localhost"`
	Mode        string `mapstructure:"mode" default:"debug"`
	Prefix      string `mapstructure:"prefix" default:"/ethereum"`
}
