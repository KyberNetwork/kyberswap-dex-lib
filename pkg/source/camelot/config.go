package camelot

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   uint   `json:"newPoolLimit"`
}
