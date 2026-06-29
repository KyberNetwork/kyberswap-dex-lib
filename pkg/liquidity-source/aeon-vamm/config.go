package aeonvamm

const (
	DexTypeAeonVAMM = "aeon-vamm"
	defaultChainID  = 43114 // Avalanche C-Chain
)

type Config struct {
	DexID          string `json:"dexId"`
	FactoryAddress string `json:"factoryAddress"`
	ChainID        int    `json:"chainId"`
}

var defaultConfig = &Config{
	DexID:          DexTypeAeonVAMM,
	FactoryAddress: "0x3ECf287990A2365d48C6681620393aC1cdF3D268",
	ChainID:        defaultChainID,
}
