package poe

type Config struct {
	DexId          string `json:"dexId"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
