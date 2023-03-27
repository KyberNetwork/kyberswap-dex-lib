package metrics

type Config struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
}
