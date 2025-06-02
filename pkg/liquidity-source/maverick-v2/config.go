package maverickv2

type Config struct {
	DexID           string `json:"dexID"`
	FactoryAddress  string `json:"factoryAddress"`
	PoolLensAddress string `json:"poolLensAddress"`
	NewPoolLimit    int    `json:"newPoolLimit"`
}
