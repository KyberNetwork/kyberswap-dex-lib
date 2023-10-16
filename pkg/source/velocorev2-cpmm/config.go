package velocorev2cpmm

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
