package poolrank

type Config struct {
	Redis RedisRepositoryConfig `mapstructure:"redis"`

	UseNativeRanking bool `mapstructure:"useNativeRanking"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}
