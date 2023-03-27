package makerpsm

type Config struct {
	Dai  DaiConfig   `json:"dai"`
	PSMs []PSMConfig `json:"psms"`
}

type PSMConfig struct {
	Address string `json:"address"`
	Gem     struct {
		Address  string `json:"address"`
		Decimals uint8  `json:"decimals"`
	} `json:"gem"`
}

type DaiConfig struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}
