package llamma

type Config struct {
	DexID               string `json:"dexID"`
	FactoryAddress      string `json:"factoryAddress"`
	StableCoin          string `json:"stableCoin"`
	NewPoolLimit        int    `json:"newPoolLimit"`
	LlammaHelperAddress string `json:"llammaHelperAddress"`
}
