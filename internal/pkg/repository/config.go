package repository

type Config struct {
	RPCRepository RPCRepositoryConfig
}

type RPCRepositoryConfig struct {
	RPCs             []string
	MulticallAddress string
}

func DefaultConfig() Config {
	return Config{
		RPCRepository: RPCRepositoryConfig{
			RPCs:             []string{},
			MulticallAddress: "",
		},
	}
}
