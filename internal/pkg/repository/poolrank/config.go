package poolrank

type Config struct {
	Redis                 RedisRepositoryConfig `mapstructure:"redis"`
	SetsNeededTobeIndexed map[string]bool       `mapstructure:"setsNeededTobeIndexed"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}
