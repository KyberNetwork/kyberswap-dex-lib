package trackexecutor

type Config struct {
	SubgraphURL       string   `mapstructure:"subgraphURL"`
	GasTokenAddress   string   `mapstructure:"gasTokenAddress"`
	ExecutorAddresses []string `mapstructure:"executorAddresses"`
}
