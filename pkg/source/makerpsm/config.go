package makerpsm

type Config struct {
	DexID      string    `json:"-"`
	ConfigPath string    `json:"configPath"`
	DexConfig  DexConfig `json:"-"`
}
