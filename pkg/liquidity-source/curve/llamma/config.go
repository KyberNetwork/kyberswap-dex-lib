package llamma

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	StableCoin     string `json:"stableCoin"`
	HelperAddress  string `json:"helperAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
