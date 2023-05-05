package pool

type Config struct {
	Redis RedisRepositoryConfig `mapstructure:"redis"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}
